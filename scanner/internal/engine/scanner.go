package engine

import (
	"bytes"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"time"

	"github.com/warbletech/kestrelops/scanner/internal/report"
	"gopkg.in/yaml.v3"
)

// ScanRequest is the input payload for a scan
type ScanRequest struct {
	Manifest string `json:"manifest"` // raw YAML/Helm content
	Source   string `json:"source"`   // filename or repo URL label
}

// Run executes all scan checks and returns a ScanReport
func Run(req ScanRequest) (*report.ScanReport, error) {
	var findings []report.Finding

	yamlFindings := runYAMLLint(req.Manifest)
	findings = append(findings, yamlFindings...)

	plutoFindings := runPluto(req.Manifest)
	findings = append(findings, plutoFindings...)

	polarisFindings := runPolaris(req.Manifest)
	findings = append(findings, polarisFindings...)

	kubeScoreFindings := runKubeScore(req.Manifest)
	findings = append(findings, kubeScoreFindings...)

	score := computeScore(findings)
	summary := buildSummary(findings)

	return &report.ScanReport{
		ScanID:    fmt.Sprintf("kop-%d", time.Now().UnixMilli()),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Source:    req.Source,
		Score:     score,
		Findings:  findings,
		Summary:   summary,
	}, nil
}

// runYAMLLint validates YAML structure using the Go yaml parser
func runYAMLLint(content string) []report.Finding {
	var findings []report.Finding
	docs := strings.Split(content, "---")
	for i, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		var node yaml.Node
		if err := yaml.Unmarshal([]byte(doc), &node); err != nil {
			findings = append(findings, report.Finding{
				ID:          fmt.Sprintf("YAML-%03d", i+1),
				Category:    report.CategoryYAML,
				Severity:    report.SeverityHigh,
				Resource:    fmt.Sprintf("document[%d]", i+1),
				Message:     fmt.Sprintf("YAML parse error: %v", err),
				Remediation: "Fix YAML syntax — check indentation, colons, and special characters",
			})
		}
	}
	return findings
}

// runPluto checks for deprecated/removed Kubernetes API versions
func runPluto(content string) []report.Finding {
	var findings []report.Finding
	plutoPath, err := exec.LookPath("pluto")
	if err != nil {
		return append(findings, report.Finding{
			ID:          "PLUTO-NOTFOUND",
			Category:    report.CategoryDeprecated,
			Severity:    report.SeverityInfo,
			Resource:    "scanner",
			Message:     "pluto binary not found — skipping deprecated API check",
			Remediation: "Install pluto: https://pluto.docs.fairwinds.com/",
		})
	}

	cmd := exec.Command(plutoPath, "detect", "-", "--output", "json", "--ignore-deprecations=false")
	cmd.Stdin = bytes.NewBufferString(content)
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return findings
	}

	// Parse pluto JSON output
	type plutoItem struct {
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		Version     string `json:"version"`
		Replacement string `json:"replacement_api"`
		Removed     bool   `json:"removed"`
		Deprecated  bool   `json:"deprecated"`
	}
	type plutoOutput struct {
		Items []plutoItem `json:"items"`
	}

	var result plutoOutput
	if err := parseJSON(out, &result); err != nil {
		return findings
	}

	for _, item := range result.Items {
		sev := report.SeverityMedium
		msg := fmt.Sprintf("Deprecated API: %s %s", item.Kind, item.Version)
		rem := fmt.Sprintf("Migrate to %s", item.Replacement)
		if item.Removed {
			sev = report.SeverityHigh
			msg = fmt.Sprintf("Removed API: %s %s — will break on upgrade", item.Kind, item.Version)
		}
		findings = append(findings, report.Finding{
			ID:          fmt.Sprintf("PLUTO-%s-%s", item.Kind, item.Name),
			Category:    report.CategoryDeprecated,
			Severity:    sev,
			Resource:    fmt.Sprintf("%s/%s", item.Kind, item.Name),
			Message:     msg,
			Remediation: rem,
			DocsURL:     "https://pluto.docs.fairwinds.com/",
		})
	}
	return findings
}

// runPolaris checks security best practices via Polaris CLI
func runPolaris(content string) []report.Finding {
	var findings []report.Finding
	polarisPath, err := exec.LookPath("polaris")
	if err != nil {
		return append(findings, report.Finding{
			ID:          "POLARIS-NOTFOUND",
			Category:    report.CategorySecurity,
			Severity:    report.SeverityInfo,
			Resource:    "scanner",
			Message:     "polaris binary not found — skipping security policy check",
			Remediation: "Install polaris: https://polaris.docs.fairwinds.com/",
		})
	}

	cmd := exec.Command(polarisPath, "audit", "--audit-path", "-", "--format", "json")
	cmd.Stdin = bytes.NewBufferString(content)
	out, _ := cmd.Output()

	type polarisResult struct {
		Results map[string]map[string]struct {
			ID       string `json:"id"`
			Message  string `json:"message"`
			Severity string `json:"severity"`
			Success  bool   `json:"success"`
		} `json:"results"`
	}

	var result polarisResult
	if err := parseJSON(out, &result); err != nil {
		return findings
	}

	for resource, checks := range result.Results {
		for _, check := range checks {
			if check.Success {
				continue
			}
			sev := report.SeverityMedium
			if check.Severity == "critical" {
				sev = report.SeverityCritical
			} else if check.Severity == "warning" {
				sev = report.SeverityLow
			}
			findings = append(findings, report.Finding{
				ID:          fmt.Sprintf("POL-%s", check.ID),
				Category:    report.CategorySecurity,
				Severity:    sev,
				Resource:    resource,
				Message:     check.Message,
				Remediation: fmt.Sprintf("See: https://polaris.docs.fairwinds.com/checks/%s", check.ID),
				DocsURL:     fmt.Sprintf("https://polaris.docs.fairwinds.com/checks/%s", check.ID),
			})
		}
	}
	return findings
}

// runKubeScore checks reliability and security scoring
func runKubeScore(content string) []report.Finding {
	var findings []report.Finding
	ksPath, err := exec.LookPath("kube-score")
	if err != nil {
		return append(findings, report.Finding{
			ID:          "KUBESCORE-NOTFOUND",
			Category:    report.CategoryReliability,
			Severity:    report.SeverityInfo,
			Resource:    "scanner",
			Message:     "kube-score binary not found — skipping reliability check",
			Remediation: "Install kube-score: https://kube-score.com/",
		})
	}

	cmd := exec.Command(ksPath, "score", "-", "--output-format", "json")
	cmd.Stdin = bytes.NewBufferString(content)
	out, _ := cmd.Output()

	type ksCheck struct {
		Name     string `json:"name"`
		Grade    int    `json:"grade"` // 0=critical,5=warning,10=ok
		Comments []struct {
			Summary  string `json:"summary"`
			Description string `json:"description"`
		} `json:"comments"`
	}
	type ksObject struct {
		ObjectName string    `json:"object_name"`
		Checks     []ksCheck `json:"checks"`
	}

	var results []ksObject
	if err := parseJSON(out, &results); err != nil {
		return findings
	}

	for _, obj := range results {
		for _, check := range obj.Checks {
			if check.Grade == 10 {
				continue
			}
			sev := report.SeverityMedium
			if check.Grade == 0 {
				sev = report.SeverityCritical
			} else if check.Grade == 5 {
				sev = report.SeverityLow
			}
			msg := check.Name
			if len(check.Comments) > 0 {
				msg = check.Comments[0].Summary
			}
			findings = append(findings, report.Finding{
				ID:          fmt.Sprintf("KS-%s-%s", obj.ObjectName, check.Name),
				Category:    report.CategoryReliability,
				Severity:    sev,
				Resource:    obj.ObjectName,
				Message:     msg,
				Remediation: "See kube-score docs: https://kube-score.com/",
				DocsURL:     "https://kube-score.com/",
			})
		}
	}
	return findings
}

// computeScore calculates the weighted KestrelOps score
func computeScore(findings []report.Finding) report.ScoreBreakdown {
	penaltyBySeverity := map[report.Severity]float64{
		report.SeverityCritical: 25,
		report.SeverityHigh:     15,
		report.SeverityMedium:   7,
		report.SeverityLow:      3,
		report.SeverityInfo:     0,
	}
	penaltyByCategory := map[report.Category]float64{
		report.CategorySecurity:    0,
		report.CategoryDeprecated:  0,
		report.CategoryReliability: 0,
		report.CategoryHelm:        0,
		report.CategoryYAML:        0,
	}

	for _, f := range findings {
		penaltyByCategory[f.Category] += penaltyBySeverity[f.Severity]
	}

	clamp := func(v float64) float64 {
		return math.Max(0, math.Min(100, 100-v))
	}

	security := clamp(penaltyByCategory[report.CategorySecurity])
	apiStability := clamp(penaltyByCategory[report.CategoryDeprecated])
	reliability := clamp(penaltyByCategory[report.CategoryReliability])
	lintClean := clamp(penaltyByCategory[report.CategoryHelm] + penaltyByCategory[report.CategoryYAML])

	overall := security*0.40 + apiStability*0.20 + reliability*0.25 + lintClean*0.15

	grade := "F"
	switch {
	case overall >= 90:
		grade = "A"
	case overall >= 75:
		grade = "B"
	case overall >= 60:
		grade = "C"
	case overall >= 45:
		grade = "D"
	}

	return report.ScoreBreakdown{
		Security:     math.Round(security*10) / 10,
		APIStability: math.Round(apiStability*10) / 10,
		Reliability:  math.Round(reliability*10) / 10,
		LintClean:    math.Round(lintClean*10) / 10,
		Overall:      math.Round(overall*10) / 10,
		Grade:        grade,
	}
}

// buildSummary counts findings by severity
func buildSummary(findings []report.Finding) report.ScanSummary {
	s := report.ScanSummary{Total: len(findings)}
	for _, f := range findings {
		switch f.Severity {
		case report.SeverityCritical:
			s.Critical++
		case report.SeverityHigh:
			s.High++
		case report.SeverityMedium:
			s.Medium++
		case report.SeverityLow:
			s.Low++
		case report.SeverityInfo:
			s.Info++
		}
	}
	return s
}
