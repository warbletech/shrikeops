package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureName maps testdata/manifests/<dir>/<name>.yaml → "<dir>-<name>".
func fixtureName(manifestPath string) string {
	dir := filepath.Base(filepath.Dir(manifestPath))
	return dir + "-" + strings.TrimSuffix(filepath.Base(manifestPath), ".yaml")
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// replayTools makes Run use the captured tool outputs for fixture `name`
// instead of executing binaries. `missing` tools behave as not on PATH.
func replayTools(t *testing.T, name string, missing ...string) {
	t.Helper()
	origLook, origRun := lookPath, runTool
	t.Cleanup(func() { lookPath, runTool = origLook, origRun })

	gone := map[string]bool{}
	for _, m := range missing {
		gone[m] = true
	}
	lookPath = func(tool string) (string, error) {
		if gone[tool] {
			return "", fmt.Errorf("exec: %q: executable file not found in $PATH", tool)
		}
		return tool, nil
	}
	runTool = func(tool string, _ []string, _ []byte) ([]byte, int, error) {
		b, err := os.ReadFile(filepath.Join("testdata", "tools", tool, name+".json"))
		if err != nil {
			return nil, 1, err
		}
		return b, 0, nil
	}
}

func manifests(t *testing.T) []string {
	t.Helper()
	m, err := filepath.Glob("testdata/manifests/*/*.yaml")
	if err != nil || len(m) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	return m
}
