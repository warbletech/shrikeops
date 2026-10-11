package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/warbletech/shrikeops/scanner/internal/report"
)

const demoBad = "../../internal/engine/testdata/manifests/bad/privileged-pod.yaml"

// Issue #14: banner must say ShrikeOps, never KestrelOps.
func TestBannerText(t *testing.T) {
	b := bannerText()
	if strings.Contains(b, "| ' / ___ ___|") || strings.Contains(strings.ToLower(b), "kestrel") {
		t.Fatal("banner still contains the KestrelOps art")
	}
	if !strings.Contains(b, "/ ___|| |__") || !strings.Contains(b, "v"+version) {
		t.Fatalf("unexpected banner:\n%s", b)
	}
}

// Issue #14: --json stdout must be pure JSON (banner never on stdout).
func TestScanJSONStdoutIsPureJSON(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no external tools → also exercises incomplete path
	var out, errb bytes.Buffer
	code := run([]string{"scan", demoBad, "--json"}, nil, &out, &errb)
	var r report.ScanReport
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out.String())
	}
	if r.Score.Complete || r.Score.Grade != report.GradeIncomplete {
		t.Fatalf("expected incomplete score, got %+v", r.Score)
	}
	if code != exitIncomplete {
		t.Fatalf("exit = %d, want %d", code, exitIncomplete)
	}
	if !strings.Contains(errb.String(), "INCOMPLETE") {
		t.Fatalf("stderr should warn, got %q", errb.String())
	}
}

func TestScanAllowIncomplete(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var out, errb bytes.Buffer
	code := run([]string{"scan", "--allow-incomplete", demoBad}, nil, &out, &errb)
	if code != exitOK {
		t.Fatalf("exit = %d, want 0", code)
	}
	s := out.String()
	if !strings.Contains(s, "INCOMPLETE") || !strings.Contains(s, "SKIPPED  polaris") || strings.Contains(s, "(A)") {
		t.Fatalf("human output should flag incomplete scan:\n%s", s)
	}
	if strings.Contains(s, "____") {
		t.Fatal("banner must not be written to stdout")
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name  string
		sum   report.ScanSummary
		full  bool
		allow bool
		want  int
	}{
		{"clean complete", report.ScanSummary{Medium: 3}, true, false, exitOK},
		{"high findings", report.ScanSummary{High: 1}, true, false, exitFindings},
		{"critical beats incomplete", report.ScanSummary{Critical: 1}, false, false, exitFindings},
		{"incomplete", report.ScanSummary{}, false, false, exitIncomplete},
		{"incomplete allowed", report.ScanSummary{}, false, true, exitOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &report.ScanReport{Summary: tc.sum, Score: report.ScoreBreakdown{Complete: tc.full}}
			if got := exitCode(r, tc.allow); got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{{"scan"}, {"bogus"}, {"scan", "a.yaml", "b.yaml"}, {"scan", "/does/not/exist.yaml"}} {
		var out, errb bytes.Buffer
		if code := run(args, nil, &out, &errb); code != exitUsage {
			t.Errorf("%v: exit %d, want %d", args, code, exitUsage)
		}
	}
}

func TestScanStdin(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	b, err := os.ReadFile(demoBad)
	if err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	run([]string{"scan", "-", "--json"}, bytes.NewReader(b), &out, &errb)
	var r report.ScanReport
	if err := json.Unmarshal(out.Bytes(), &r); err != nil || r.Source != "stdin" {
		t.Fatalf("stdin scan failed: %v %q", err, r.Source)
	}
}
