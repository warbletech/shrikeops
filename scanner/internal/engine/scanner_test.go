package engine

import (
	"os/exec"
	"reflect"
	"testing"

	"github.com/warbletech/shrikeops/scanner/internal/report"
)

func TestComputeScore(t *testing.T) {
	sec := func(s report.Severity) report.Finding {
		return report.Finding{Category: report.CategorySecurity, Severity: s}
	}
	tests := []struct {
		name     string
		findings []report.Finding
		skipped  []report.SkippedCheck
		want     report.ScoreBreakdown
	}{
		{
			name: "clean",
			want: report.ScoreBreakdown{Security: 100, APIStability: 100, Reliability: 100, LintClean: 100, Overall: 100, Grade: "A", Complete: true},
		},
		{
			name:     "one high security finding",
			findings: []report.Finding{sec(report.SeverityHigh)},
			want:     report.ScoreBreakdown{Security: 85, APIStability: 100, Reliability: 100, LintClean: 100, Overall: 94, Grade: "A", Complete: true},
		},
		{
			name:     "security clamps at zero",
			findings: []report.Finding{sec(report.SeverityCritical), sec(report.SeverityCritical), sec(report.SeverityCritical), sec(report.SeverityCritical), sec(report.SeverityCritical)},
			want:     report.ScoreBreakdown{Security: 0, APIStability: 100, Reliability: 100, LintClean: 100, Overall: 60, Grade: "C", Complete: true},
		},
		{
			name:     "info and scanner findings do not penalise",
			findings: []report.Finding{sec(report.SeverityInfo), {Category: report.CategoryScanner, Severity: report.SeverityInfo}},
			want:     report.ScoreBreakdown{Security: 100, APIStability: 100, Reliability: 100, LintClean: 100, Overall: 100, Grade: "A", Complete: true},
		},
		{
			name:     "helm lint counts toward lint clean",
			findings: []report.Finding{{Category: report.CategoryHelm, Severity: report.SeverityHigh}},
			want:     report.ScoreBreakdown{Security: 100, APIStability: 100, Reliability: 100, LintClean: 85, Overall: 97.8, Grade: "A", Complete: true},
		},
		{
			// Issue #10: a missing tool must not count as a pass.
			name:    "missing polaris → incomplete, N/A, re-weighted overall",
			skipped: []report.SkippedCheck{{Tool: "polaris", Category: report.CategorySecurity}},
			findings: []report.Finding{
				{Category: report.CategoryReliability, Severity: report.SeverityCritical},
			},
			// (100*0.20 + 75*0.25 + 100*0.15) / 0.60 = 89.58
			want: report.ScoreBreakdown{Security: 0, APIStability: 100, Reliability: 75, LintClean: 100, Overall: 89.6, Grade: report.GradeIncomplete, Complete: false},
		},
		{
			name: "all external tools missing",
			skipped: []report.SkippedCheck{
				{Tool: "pluto", Category: report.CategoryDeprecated},
				{Tool: "polaris", Category: report.CategorySecurity},
				{Tool: "kube-score", Category: report.CategoryReliability},
			},
			want: report.ScoreBreakdown{LintClean: 100, Overall: 100, Grade: report.GradeIncomplete, Complete: false},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeScore(tc.findings, tc.skipped)
			got.SkippedChecks = nil
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got  %+v\nwant %+v", got, tc.want)
			}
		})
	}
}

// End-to-end scoring of the shrikeops-demo manifests using recorded tool output.
func TestRunDemoManifests(t *testing.T) {
	tests := []struct {
		fixture string
		overall float64
		grade   string
		// security, api stability, reliability
		sec, api, rel float64
		high          int
	}{
		{"good-rbac", 100, "A", 100, 100, 100, 0},
		{"good-service", 98.3, "A", 100, 100, 93, 0},
		{"good-networkpolicy", 98.3, "A", 100, 100, 93, 0},
		{"good-deployment", 89.3, "B", 97, 100, 62, 0},
		{"bad-root-container", 65.5, "C", 40, 100, 58, 3},
		{"bad-deprecated-api", 63.6, "C", 52, 70, 55, 4},
		{"bad-privileged-pod", 47.8, "D", 0, 100, 51, 6},
		{"bad-no-limits", 45.7, "D", 1, 100, 41, 5},
	}
	for _, tc := range tests {
		t.Run(tc.fixture, func(t *testing.T) {
			replayTools(t, tc.fixture)
			dir, name := "good", tc.fixture[len("good-"):]
			if tc.fixture[:3] == "bad" {
				dir, name = "bad", tc.fixture[len("bad-"):]
			}
			r, err := Run(ScanRequest{Manifest: string(readFile(t, "testdata/manifests/"+dir+"/"+name+".yaml")), Source: name})
			if err != nil {
				t.Fatal(err)
			}
			s := r.Score
			if !s.Complete || len(s.SkippedChecks) != 0 {
				t.Fatalf("expected complete scan, got %+v", s)
			}
			if s.Overall != tc.overall || s.Grade != tc.grade || s.Security != tc.sec || s.APIStability != tc.api || s.Reliability != tc.rel {
				t.Errorf("got overall=%v grade=%s sec=%v api=%v rel=%v", s.Overall, s.Grade, s.Security, s.APIStability, s.Reliability)
			}
			if r.Summary.High != tc.high || r.Summary.Critical != 0 {
				t.Errorf("got high=%d critical=%d, want high=%d", r.Summary.High, r.Summary.Critical, tc.high)
			}
			// good manifests must never score below B; bad ones never A/B.
			if dir == "good" && s.Overall < 75 {
				t.Errorf("good manifest scored %v", s.Overall)
			}
			if dir == "bad" && s.Overall >= 75 {
				t.Errorf("bad manifest scored %v", s.Overall)
			}
		})
	}
}

// Issue #10: missing tools → incomplete report, never 100/A.
func TestRunMissingTools(t *testing.T) {
	m := string(readFile(t, "testdata/manifests/bad/privileged-pod.yaml"))
	tests := []struct {
		name    string
		missing []string
	}{
		{"all missing", []string{"pluto", "polaris", "kube-score"}},
		{"polaris missing", []string{"polaris"}},
		{"kube-score missing", []string{"kube-score"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			replayTools(t, "bad-privileged-pod", tc.missing...)
			r, _ := Run(ScanRequest{Manifest: m, Source: "privileged-pod.yaml"})
			if r.Score.Complete || r.Score.Grade != report.GradeIncomplete {
				t.Fatalf("expected incomplete, got %+v", r.Score)
			}
			if len(r.Score.SkippedChecks) != len(tc.missing) {
				t.Fatalf("skipped = %+v", r.Score.SkippedChecks)
			}
			for i, s := range r.Score.SkippedChecks {
				if s.Reason != "not found on PATH" || s.Install == "" {
					t.Errorf("skipped[%d] = %+v", i, s)
				}
			}
			n := 0
			for _, f := range r.Findings {
				if f.Category == report.CategoryScanner {
					n++
				}
			}
			if n != len(tc.missing) {
				t.Errorf("want %d scanner findings, got %d", len(tc.missing), n)
			}
		})
	}
}

func TestRunToolFailureIsIncomplete(t *testing.T) {
	replayTools(t, "good-rbac")
	runTool = func(tool string, _ []string, _ []byte) ([]byte, int, error) {
		if tool == "polaris" {
			return nil, 1, &exec.ExitError{}
		}
		return []byte(`{"items":[]}`), 0, nil
	}
	// kube-score gets pluto-shaped output → unparseable → also skipped.
	r, _ := Run(ScanRequest{Manifest: "apiVersion: v1\nkind: Namespace\nmetadata: {name: x}\n"})
	if r.Score.Complete {
		t.Fatal("tool failure must make the scan incomplete")
	}
	got := map[string]bool{}
	for _, s := range r.Score.SkippedChecks {
		got[s.Tool] = true
	}
	if !got["polaris"] || !got["kube-score"] || got["pluto"] {
		t.Fatalf("skipped = %+v", r.Score.SkippedChecks)
	}
}

func TestRunYAMLLint(t *testing.T) {
	replayTools(t, "good-rbac")
	r, _ := Run(ScanRequest{Manifest: "a: [1, 2\n---\nb: ok\n"})
	if r.Summary.High < 1 || r.Score.LintClean != 85 {
		t.Fatalf("want yaml finding, got %+v %+v", r.Summary, r.Score)
	}
}

// Runs the real binaries when they are installed (CI installs them).
func TestRunRealToolsIfInstalled(t *testing.T) {
	for _, tool := range []string{"pluto", "polaris", "kube-score"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}
	for _, m := range manifests(t) {
		t.Run(fixtureName(m), func(t *testing.T) {
			r, err := Run(ScanRequest{Manifest: string(readFile(t, m)), Source: m})
			if err != nil {
				t.Fatal(err)
			}
			if !r.Score.Complete {
				t.Fatalf("incomplete with tools installed: %+v", r.Score.SkippedChecks)
			}
			for _, f := range r.Findings {
				if f.Message == "" {
					t.Errorf("empty message: %+v", f)
				}
			}
		})
	}
}
