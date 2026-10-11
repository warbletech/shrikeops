package engine

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/warbletech/shrikeops/scanner/internal/report"
	"gopkg.in/yaml.v3"
)

// ScanRequest is the input payload for a scan
type ScanRequest struct {
	Manifest string `json:"manifest"` // raw YAML/Helm content
	Source   string `json:"source"`   // filename or repo URL label
}

// externalTool describes one third-party scanner ShrikeOps shells out to.
type externalTool struct {
	Name     string
	Category report.Category
	Install  string
	// args builds the argv; manifestPath is set only when NeedsFile is true.
	Args      func(manifestPath string) []string
	NeedsFile bool // tool cannot read the manifest from stdin
	// okExit reports whether a non-zero exit code is a normal "findings found" result.
	OkExit func(code int) bool
	Parse  func(out []byte) ([]report.Finding, error)
}

// Tools is the set of external checks. Exported for docs/tests.
var Tools = []externalTool{
	{
		Name:     "pluto",
		Category: report.CategoryDeprecated,
		Install:  "https://github.com/FairwindsOps/pluto/releases (or: brew install FairwindsOps/tap/pluto)",
		Args: func(string) []string {
			return []string{"detect", "-", "--output", "json", "--ignore-deprecations=false"}
		},
		// pluto exits 2 when deprecated and 3 when removed APIs are found.
		OkExit: func(code int) bool { return code == 2 || code == 3 },
		Parse:  parsePluto,
	},
	{
		Name:     "polaris",
		Category: report.CategorySecurity,
		Install:  "https://github.com/FairwindsOps/polaris/releases (or: brew install FairwindsOps/tap/polaris)",
		// polaris does not accept "-" for --audit-path; it needs a real file.
		NeedsFile: true,
		Args: func(path string) []string {
			return []string{"audit", "--audit-path", path, "--format", "json"}
		},
		OkExit: func(int) bool { return false },
		Parse:  parsePolaris,
	},
	{
		Name:     "kube-score",
		Category: report.CategoryReliability,
		Install:  "https://github.com/zegl/kube-score/releases (or: brew install kube-score)",
		Args: func(string) []string {
			return []string{"score", "-", "--output-format", "json"}
		},
		// kube-score exits 1 when it found critical issues.
		OkExit: func(code int) bool { return code == 1 },
		Parse:  parseKubeScore,
	},
}

// Test seams.
var (
	lookPath = exec.LookPath
	runTool  = func(path string, args []string, stdin []byte) ([]byte, int, error) {
		cmd := exec.Command(path, args...)
		cmd.Stdin = bytes.NewReader(stdin)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if msg := strings.TrimSpace(stderr.String()); msg != "" {
				err = fmt.Errorf("%w: %s", err, firstLine(msg))
			}
			return out, exitErr.ExitCode(), err
		}
		return out, 0, err
	}
)

// Run executes all scan checks and returns a ScanReport
func Run(req ScanRequest) (*report.ScanReport, error) {
	findings := runYAMLLint(req.Manifest)
	var skipped []report.SkippedCheck

	for _, t := range Tools {
		f, sk := runExternal(t, req.Manifest)
		findings = append(findings, f...)
		if sk != nil {
			skipped = append(skipped, *sk)
			findings = append(findings, report.Finding{
				ID:          strings.ToUpper(strings.ReplaceAll(t.Name, "-", "")) + "-NOTRUN",
				Category:    report.CategoryScanner,
				Severity:    report.SeverityInfo,
				Resource:    "scanner",
				Message:     fmt.Sprintf("%s check did not run (%s) — %s score not computed, scan is INCOMPLETE", t.Name, sk.Reason, t.Category),
				Remediation: "Install " + t.Name + ": " + t.Install,
			})
		}
	}

	sortFindings(findings)

	return &report.ScanReport{
		ScanID:    fmt.Sprintf("kop-%d", time.Now().UnixMilli()),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Source:    req.Source,
		Score:     computeScore(findings, skipped),
		Findings:  findings,
		Summary:   buildSummary(findings),
	}, nil
}

// runExternal runs one external tool. It returns a non-nil SkippedCheck when
// the tool is missing or its output could not be used.
func runExternal(t externalTool, manifest string) ([]report.Finding, *report.SkippedCheck) {
	skip := func(reason string) *report.SkippedCheck {
		return &report.SkippedCheck{Tool: t.Name, Category: t.Category, Reason: reason, Install: t.Install}
	}
	path, err := lookPath(t.Name)
	if err != nil {
		return nil, skip("not found on PATH")
	}

	var stdin []byte
	var args []string
	if t.NeedsFile {
		f, err := os.CreateTemp("", "shrikeops-*.yaml")
		if err != nil {
			return nil, skip("cannot create temp file: " + err.Error())
		}
		defer os.Remove(f.Name())
		if _, err := f.WriteString(manifest); err != nil {
			f.Close()
			return nil, skip("cannot write temp file: " + err.Error())
		}
		f.Close()
		args = t.Args(f.Name())
	} else {
		stdin = []byte(manifest)
		args = t.Args("")
	}

	out, code, err := runTool(path, args, stdin)
	if err != nil && !(code != 0 && t.OkExit(code)) {
		return nil, skip(fmt.Sprintf("failed: %v", err))
	}
	findings, perr := t.Parse(out)
	if perr != nil {
		return nil, skip(fmt.Sprintf("unparseable output: %v", perr))
	}
	return findings, nil
}

// runYAMLLint validates YAML structure using the Go yaml parser
func runYAMLLint(content string) []report.Finding {
	var findings []report.Finding
	docs := strings.Split(content, "\n---")
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

// categoryWeights are the documented ShrikeOps score weights.
var categoryWeights = map[report.Category]float64{
	report.CategorySecurity:    0.40,
	report.CategoryDeprecated:  0.20,
	report.CategoryReliability: 0.25,
	report.CategoryYAML:        0.15, // "lint clean" = helm + yaml
}

// computeScore calculates the weighted ShrikeOps score. Categories whose tool
// was skipped are not scored: they are reported as 0, excluded from Overall
// (re-weighted over what did run) and the grade becomes N/A.
func computeScore(findings []report.Finding, skipped []report.SkippedCheck) report.ScoreBreakdown {
	penaltyBySeverity := map[report.Severity]float64{
		report.SeverityCritical: 25,
		report.SeverityHigh:     15,
		report.SeverityMedium:   7,
		report.SeverityLow:      3,
		report.SeverityInfo:     0,
	}
	penalty := map[report.Category]float64{}
	for _, f := range findings {
		cat := f.Category
		if cat == report.CategoryHelm {
			cat = report.CategoryYAML
		}
		penalty[cat] += penaltyBySeverity[f.Severity]
	}

	notRun := map[report.Category]bool{}
	for _, s := range skipped {
		notRun[s.Category] = true
	}

	scores := map[report.Category]float64{}
	var weighted, weightSum float64
	for cat, w := range categoryWeights {
		if notRun[cat] {
			scores[cat] = 0
			continue
		}
		v := math.Max(0, math.Min(100, 100-penalty[cat]))
		scores[cat] = v
		weighted += v * w
		weightSum += w
	}
	overall := 0.0
	if weightSum > 0 {
		overall = weighted / weightSum
	}

	grade := gradeFor(overall)
	complete := len(skipped) == 0
	if !complete {
		grade = report.GradeIncomplete
	}

	r1 := func(v float64) float64 { return math.Round(v*10) / 10 }
	return report.ScoreBreakdown{
		Security:      r1(scores[report.CategorySecurity]),
		APIStability:  r1(scores[report.CategoryDeprecated]),
		Reliability:   r1(scores[report.CategoryReliability]),
		LintClean:     r1(scores[report.CategoryYAML]),
		Overall:       r1(overall),
		Grade:         grade,
		Complete:      complete,
		SkippedChecks: skipped,
	}
}

func gradeFor(overall float64) string {
	switch {
	case overall >= 90:
		return "A"
	case overall >= 75:
		return "B"
	case overall >= 60:
		return "C"
	case overall >= 45:
		return "D"
	}
	return "F"
}

var severityRank = map[report.Severity]int{
	report.SeverityCritical: 0,
	report.SeverityHigh:     1,
	report.SeverityMedium:   2,
	report.SeverityLow:      3,
	report.SeverityInfo:     4,
}

// sortFindings orders findings by severity, then resource, then id, so output
// is stable (polaris results come from maps).
func sortFindings(f []report.Finding) {
	sort.SliceStable(f, func(i, j int) bool {
		if severityRank[f[i].Severity] != severityRank[f[j].Severity] {
			return severityRank[f[i].Severity] < severityRank[f[j].Severity]
		}
		if f[i].Resource != f[j].Resource {
			return f[i].Resource < f[j].Resource
		}
		return f[i].ID < f[j].ID
	})
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

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
