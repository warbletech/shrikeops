package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/warbletech/shrikeops/scanner/internal/report"
)

// ── pluto ────────────────────────────────────────────────────────────────────

type plutoOutput struct {
	Items []struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		API       struct {
			Version                string `json:"version"`
			Kind                   string `json:"kind"`
			DeprecatedIn           string `json:"deprecated-in"`
			RemovedIn              string `json:"removed-in"`
			ReplacementAPI         string `json:"replacement-api"`
			ReplacementAvailableIn string `json:"replacement-available-in"`
		} `json:"api"`
		Deprecated bool `json:"deprecated"`
		Removed    bool `json:"removed"`
	} `json:"items"`
}

// parsePluto converts `pluto detect --output json` output into findings.
func parsePluto(out []byte) ([]report.Finding, error) {
	var res plutoOutput
	if err := json.Unmarshal(bytes.TrimSpace(out), &res); err != nil {
		return nil, err
	}
	var findings []report.Finding
	for _, it := range res.Items {
		api := it.API
		resource := joinNonEmpty("/", api.Kind, it.Namespace, it.Name)
		sev := report.SeverityMedium
		msg := fmt.Sprintf("Deprecated API: %s %s", api.Kind, api.Version)
		if api.DeprecatedIn != "" {
			msg += fmt.Sprintf(" (deprecated in %s", api.DeprecatedIn)
			if api.RemovedIn != "" {
				msg += fmt.Sprintf(", removed in %s", api.RemovedIn)
			}
			msg += ")"
		}
		if it.Removed {
			sev = report.SeverityHigh
			msg = fmt.Sprintf("Removed API: %s %s", api.Kind, api.Version)
			if api.RemovedIn != "" {
				msg += fmt.Sprintf(" (removed in Kubernetes %s)", api.RemovedIn)
			}
			msg += " — will break on upgrade"
		}
		rem := "No direct replacement API; see the Kubernetes deprecation guide"
		if api.ReplacementAPI != "" {
			rem = fmt.Sprintf("Migrate to %s", api.ReplacementAPI)
			if api.ReplacementAvailableIn != "" {
				rem += fmt.Sprintf(" (available since %s)", api.ReplacementAvailableIn)
			}
		}
		findings = append(findings, report.Finding{
			ID:          fmt.Sprintf("PLUTO-%s-%s", api.Kind, it.Name),
			Category:    report.CategoryDeprecated,
			Severity:    sev,
			Resource:    resource,
			Message:     msg,
			Remediation: rem,
			DocsURL:     "https://kubernetes.io/docs/reference/using-api/deprecation-guide/",
		})
	}
	return findings, nil
}

// ── polaris ──────────────────────────────────────────────────────────────────

type polarisCheck struct {
	ID       string `json:"ID"`
	Message  string `json:"Message"`
	Success  bool   `json:"Success"`
	Severity string `json:"Severity"` // danger | warning | ignore
	Category string `json:"Category"` // Security | Reliability | Efficiency
}

type polarisOutput struct {
	Results []struct {
		Name      string                  `json:"Name"`
		Namespace string                  `json:"Namespace"`
		Kind      string                  `json:"Kind"`
		Results   map[string]polarisCheck `json:"Results"`
		PodResult *struct {
			Results          map[string]polarisCheck `json:"Results"`
			ContainerResults []struct {
				Name    string                  `json:"Name"`
				Results map[string]polarisCheck `json:"Results"`
			} `json:"ContainerResults"`
		} `json:"PodResult"`
	} `json:"Results"`
}

// parsePolaris converts `polaris audit --format json` output into findings.
// Only Security-category checks are reported: reliability is scored by
// kube-score, and counting Polaris' reliability/efficiency checks too would
// double-penalise the same issue.
func parsePolaris(out []byte) ([]report.Finding, error) {
	var res polarisOutput
	if err := json.Unmarshal(bytes.TrimSpace(out), &res); err != nil {
		return nil, err
	}
	var findings []report.Finding
	add := func(resource string, checks map[string]polarisCheck) {
		ids := make([]string, 0, len(checks))
		for id := range checks {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			c := checks[id]
			if c.Success || !strings.EqualFold(c.Category, "Security") {
				continue
			}
			var sev report.Severity
			switch strings.ToLower(c.Severity) {
			case "danger":
				sev = report.SeverityHigh
			case "warning":
				sev = report.SeverityLow
			default: // "ignore" or unknown
				continue
			}
			if c.ID == "" {
				c.ID = id
			}
			findings = append(findings, report.Finding{
				ID:          "POL-" + c.ID,
				Category:    report.CategorySecurity,
				Severity:    sev,
				Resource:    resource,
				Message:     c.Message,
				Remediation: "See https://polaris.docs.fairwinds.com/checks/security/",
				DocsURL:     "https://polaris.docs.fairwinds.com/checks/security/",
			})
		}
	}
	for _, r := range res.Results {
		base := joinNonEmpty("/", r.Kind, r.Namespace, r.Name)
		add(base, r.Results)
		if r.PodResult != nil {
			add(base, r.PodResult.Results)
			for _, c := range r.PodResult.ContainerResults {
				add(base+"/container:"+c.Name, c.Results)
			}
		}
	}
	return findings, nil
}

// ── kube-score ───────────────────────────────────────────────────────────────

// kube-score grade constants (scorecard.Grade).
const (
	ksGradeCritical = 1
	ksGradeWarning  = 5
	ksGradeAlmostOK = 7
	ksGradeAllOK    = 10
)

type kubeScoreObject struct {
	ObjectName string `json:"object_name"`
	Checks     []struct {
		Check struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"check"`
		Grade    int  `json:"grade"`
		Skipped  bool `json:"skipped"`
		Comments []struct {
			Path        string `json:"path"`
			Summary     string `json:"summary"`
			Description string `json:"description"`
		} `json:"comments"`
	} `json:"checks"`
}

// parseKubeScore converts `kube-score score --output-format json` output into
// findings. Skipped checks are not findings and are ignored. kube-score
// CRITICAL maps to medium (many are context-dependent, e.g. "no matching
// NetworkPolicy" when scanning a single file), WARNING to low.
func parseKubeScore(out []byte) ([]report.Finding, error) {
	trimmed := bytes.TrimSpace(out)
	var objs []kubeScoreObject
	if err := json.Unmarshal(trimmed, &objs); err != nil {
		return nil, err
	}
	var findings []report.Finding
	for _, obj := range objs {
		for _, c := range obj.Checks {
			if c.Skipped || c.Grade >= ksGradeAllOK {
				continue
			}
			var sev report.Severity
			switch {
			case c.Grade <= ksGradeCritical:
				sev = report.SeverityMedium
			case c.Grade <= ksGradeWarning:
				sev = report.SeverityLow
			case c.Grade <= ksGradeAlmostOK:
				sev = report.SeverityInfo
			default:
				continue
			}
			var parts []string
			for _, cm := range c.Comments {
				s := cm.Summary
				if cm.Path != "" {
					s = cm.Path + ": " + s
				}
				parts = append(parts, s)
			}
			msg := c.Check.Name
			if len(parts) > 0 {
				msg = strings.Join(parts, "; ")
			}
			id := c.Check.ID
			if id == "" {
				id = c.Check.Name
			}
			findings = append(findings, report.Finding{
				ID:          fmt.Sprintf("KS-%s-%s", obj.ObjectName, id),
				Category:    report.CategoryReliability,
				Severity:    sev,
				Resource:    obj.ObjectName,
				Message:     msg,
				Remediation: fmt.Sprintf("kube-score check %q — see https://github.com/zegl/kube-score/blob/master/README_CHECKS.md", c.Check.Name),
				DocsURL:     "https://github.com/zegl/kube-score/blob/master/README_CHECKS.md",
			})
		}
	}
	return findings, nil
}

func joinNonEmpty(sep string, parts ...string) string {
	var keep []string
	for _, p := range parts {
		if p != "" {
			keep = append(keep, p)
		}
	}
	return strings.Join(keep, sep)
}
