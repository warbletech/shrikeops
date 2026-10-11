package engine

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/warbletech/shrikeops/scanner/internal/report"
)

func toolOut(t *testing.T, tool, fixture string) []byte {
	t.Helper()
	return readFile(t, filepath.Join("testdata", "tools", tool, fixture+".json"))
}

func countBy(f []report.Finding) map[report.Severity]int {
	m := map[report.Severity]int{}
	for _, x := range f {
		m[x.Severity]++
	}
	return m
}

// Issue #12: pluto findings must carry kind/version/replacement.
func TestParsePluto(t *testing.T) {
	tests := []struct {
		fixture  string
		wantHigh int
		wantMsgs []string
		wantRems []string
		wantRes  []string
	}{
		{fixture: "good-deployment"},
		{fixture: "good-rbac"},
		{fixture: "bad-privileged-pod"},
		{
			fixture:  "bad-deprecated-api",
			wantHigh: 2,
			wantMsgs: []string{
				"Removed API: Ingress extensions/v1beta1 (removed in Kubernetes v1.22.0) — will break on upgrade",
				"Removed API: Deployment apps/v1beta1 (removed in Kubernetes v1.16.0) — will break on upgrade",
			},
			wantRems: []string{"Migrate to networking.k8s.io/v1 (available since v1.19.0)", "Migrate to apps/v1 (available since v1.9.0)"},
			wantRes:  []string{"Ingress/default/legacy-ingress", "Deployment/default/legacy-deployment"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.fixture, func(t *testing.T) {
			got, err := parsePluto(toolOut(t, "pluto", tc.fixture))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != tc.wantHigh || countBy(got)[report.SeverityHigh] != tc.wantHigh {
				t.Fatalf("got %d findings %v, want %d high", len(got), countBy(got), tc.wantHigh)
			}
			for i, f := range got {
				if f.Message != tc.wantMsgs[i] || f.Remediation != tc.wantRems[i] || f.Resource != tc.wantRes[i] {
					t.Errorf("finding %d = %q / %q / %q", i, f.Resource, f.Message, f.Remediation)
				}
				if f.Category != report.CategoryDeprecated {
					t.Errorf("category = %s", f.Category)
				}
			}
		})
	}
}

func TestParsePlutoDeprecatedOnly(t *testing.T) {
	out := []byte(`{"items":[{"name":"x","namespace":"ns","api":{"version":"policy/v1beta1","kind":"PodDisruptionBudget","deprecated-in":"v1.21.0","removed-in":"v1.25.0","replacement-api":"policy/v1"},"deprecated":true,"removed":false}]}`)
	got, err := parsePluto(out)
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v, %v", got, err)
	}
	f := got[0]
	if f.Severity != report.SeverityMedium || f.Message != "Deprecated API: PodDisruptionBudget policy/v1beta1 (deprecated in v1.21.0, removed in v1.25.0)" || f.Remediation != "Migrate to policy/v1" {
		t.Fatalf("unexpected %+v", f)
	}
}

func TestParsePlutoInvalid(t *testing.T) {
	if _, err := parsePluto([]byte("Error checking for versions: yaml: line 1")); err == nil {
		t.Fatal("expected error for non-JSON output")
	}
}

// Issue #11: polaris danger checks must surface as high security findings.
func TestParsePolaris(t *testing.T) {
	tests := []struct {
		fixture    string
		wantHigh   int
		wantLowMin int
		mustHave   []string // messages
	}{
		{fixture: "good-rbac"},
		{fixture: "good-service"},
		{fixture: "good-deployment", wantLowMin: 1},
		{fixture: "bad-privileged-pod", wantHigh: 6, wantLowMin: 1, mustHave: []string{
			"Host PID should not be configured",
			"Host IPC should not be configured",
			"Host network should not be configured",
			"Should not be running as privileged",
		}},
		{fixture: "bad-root-container", wantHigh: 3, mustHave: []string{"Should not be allowed to run as root"}},
		{fixture: "bad-no-limits", wantHigh: 5, mustHave: []string{"The ClusterRole allows Pods/exec or pods/attach"}},
	}
	for _, tc := range tests {
		t.Run(tc.fixture, func(t *testing.T) {
			got, err := parsePolaris(toolOut(t, "polaris", tc.fixture))
			if err != nil {
				t.Fatal(err)
			}
			c := countBy(got)
			if c[report.SeverityHigh] != tc.wantHigh {
				t.Errorf("high = %d, want %d", c[report.SeverityHigh], tc.wantHigh)
			}
			if c[report.SeverityLow] < tc.wantLowMin {
				t.Errorf("low = %d, want >= %d", c[report.SeverityLow], tc.wantLowMin)
			}
			for _, f := range got {
				if f.Category != report.CategorySecurity || f.Message == "" || f.Resource == "" {
					t.Errorf("bad finding %+v", f)
				}
			}
			for _, m := range tc.mustHave {
				found := false
				for _, f := range got {
					found = found || f.Message == m
				}
				if !found {
					t.Errorf("missing finding %q", m)
				}
			}
		})
	}
}

func TestParsePolarisIgnoresNonSecurityAndSuccess(t *testing.T) {
	out := []byte(`{"Results":[{"Name":"p","Namespace":"d","Kind":"Pod","Results":{},"PodResult":{"Results":{
	  "a":{"ID":"a","Message":"sec fail","Success":false,"Severity":"danger","Category":"Security"},
	  "b":{"ID":"b","Message":"sec ok","Success":true,"Severity":"danger","Category":"Security"},
	  "c":{"ID":"c","Message":"rel fail","Success":false,"Severity":"danger","Category":"Reliability"},
	  "d":{"ID":"d","Message":"ignored","Success":false,"Severity":"ignore","Category":"Security"}},
	  "ContainerResults":[{"Name":"app","Results":{"e":{"ID":"e","Message":"warn","Success":false,"Severity":"warning","Category":"Security"}}}]}}]}`)
	got, err := parsePolaris(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Message != "sec fail" || got[0].Severity != report.SeverityHigh ||
		got[1].Resource != "Pod/d/p/container:app" || got[1].Severity != report.SeverityLow {
		t.Fatalf("unexpected %+v", got)
	}
}

func TestParsePolarisInvalid(t *testing.T) {
	if _, err := parsePolaris(nil); err == nil {
		t.Fatal("expected error on empty output (e.g. polaris failed)")
	}
}

// Issue #13: skipped kube-score checks must not become findings.
func TestParseKubeScore(t *testing.T) {
	tests := []struct {
		fixture    string
		wantMedium int
		wantLow    int
	}{
		{"good-rbac", 0, 0},
		{"good-networkpolicy", 1, 0},
		{"good-service", 1, 0},
		{"good-deployment", 5, 1},
		{"bad-root-container", 6, 0},
		{"bad-no-limits", 8, 1},
		{"bad-privileged-pod", 7, 0},
	}
	for _, tc := range tests {
		t.Run(tc.fixture, func(t *testing.T) {
			got, err := parseKubeScore(toolOut(t, "kube-score", tc.fixture))
			if err != nil {
				t.Fatal(err)
			}
			c := countBy(got)
			if c[report.SeverityCritical] != 0 || c[report.SeverityHigh] != 0 {
				t.Errorf("kube-score must not produce critical/high: %v", c)
			}
			if c[report.SeverityMedium] != tc.wantMedium || c[report.SeverityLow] != tc.wantLow {
				t.Errorf("got %v, want medium=%d low=%d", c, tc.wantMedium, tc.wantLow)
			}
			for _, f := range got {
				if strings.HasPrefix(f.Message, "Skipped") || strings.Contains(f.Message, "skipping") {
					t.Errorf("skipped check reported: %+v", f)
				}
				if f.Message == "" || f.Resource == "" {
					t.Errorf("empty finding: %+v", f)
				}
			}
		})
	}
}

func TestParseKubeScoreGrades(t *testing.T) {
	out := []byte(`[{"object_name":"Deployment/apps/v1/ns/x","checks":[
	 {"check":{"id":"a","name":"A"},"grade":1,"skipped":false,"comments":[{"summary":"crit","path":"app"}]},
	 {"check":{"id":"b","name":"B"},"grade":5,"skipped":false,"comments":[{"summary":"warn"}]},
	 {"check":{"id":"c","name":"C"},"grade":7,"skipped":false,"comments":[]},
	 {"check":{"id":"d","name":"D"},"grade":10,"skipped":false},
	 {"check":{"id":"e","name":"E"},"grade":0,"skipped":true,"comments":[{"summary":"Skipped as the Deployment is not targeted by service"}]},
	 {"check":{"id":"f","name":"F"},"grade":5,"skipped":true,"comments":[{"summary":"Skipped because x is ignored"}]}]}]`)
	got, err := parseKubeScore(out)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		sev report.Severity
		msg string
	}{{report.SeverityMedium, "app: crit"}, {report.SeverityLow, "warn"}, {report.SeverityInfo, "C"}}
	if len(got) != len(want) {
		t.Fatalf("got %d findings: %+v", len(got), got)
	}
	for i, w := range want {
		if got[i].Severity != w.sev || got[i].Message != w.msg {
			t.Errorf("%d: got %s %q, want %s %q", i, got[i].Severity, got[i].Message, w.sev, w.msg)
		}
	}
}

func TestParseKubeScoreNull(t *testing.T) {
	got, err := parseKubeScore([]byte("null\n"))
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v %v", got, err)
	}
}
