package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/warbletech/kestrelops/scanner/internal/engine"
	"github.com/warbletech/kestrelops/scanner/internal/report"
)

const version = "0.1.0"

const banner = `
 _  __        _           _  ___
| |/ /       | |         | |/ _ \
| ' / ___ ___| |_ _ __ ___| | | | |_ __  ___
|  < / _ / __| __| '__/ _ | | | | | '_ \/ __|
| . \  __\__ | |_| | |  __/ | |_| | |_) \__ \
|_|\_\___|___/\__|_|  \___|_|\___/| .__/|___/
                                   | |
  Kubernetes Helm Safety Gate      |_|  v` + version + `
`

func main() {
	fmt.Print(banner)

	args := os.Args[1:]
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp()
		os.Exit(0)
	}
	if args[0] == "version" {
		fmt.Printf("kestrel version %s\n", version)
		os.Exit(0)
	}
	if args[0] != "scan" {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		printHelp()
		os.Exit(1)
	}

	// scan command
	scanArgs := args[1:]
	if len(scanArgs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: kestrel scan <file|-> [--json]")
		os.Exit(1)
	}

	jsonOutput := false
	path := ""
	for _, a := range scanArgs {
		if a == "--json" {
			jsonOutput = true
		} else {
			path = a
		}
	}

	var content []byte
	var err error
	if path == "-" {
		buf := new(bytes.Buffer)
		buf.ReadFrom(os.Stdin)
		content = buf.Bytes()
		path = "stdin"
	} else {
		content, err = os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", path, err)
			os.Exit(1)
		}
	}

	result, err := engine.Run(engine.ScanRequest{
		Manifest: string(content),
		Source:   filepath.Base(path),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	} else {
		printHuman(result)
	}

	if result.Summary.Critical > 0 || result.Summary.High > 0 {
		os.Exit(2)
	}
}

func printHuman(r *report.ScanReport) {
	grade := r.Score.Grade
	gradeColor := colorForGrade(grade)
	fmt.Printf("\n%s KestrelOps Scan — %s%s\n", bold("📋"), r.Source, reset())
	fmt.Printf("  Scan ID : %s\n", r.ScanID)
	fmt.Printf("  Score   : %s%.1f/100 (%s)%s\n", gradeColor, r.Score.Overall, grade, reset())
	fmt.Printf("  Security     : %.1f   API Stability: %.1f\n", r.Score.Security, r.Score.APIStability)
	fmt.Printf("  Reliability  : %.1f   Lint Clean   : %.1f\n\n", r.Score.Reliability, r.Score.LintClean)

	if r.Summary.Total == 0 {
		fmt.Println(green("  ✓ No issues found — chart looks healthy!"))
		return
	}

	fmt.Printf("  %s%d critical%s  %s%d high%s  %d medium  %d low  %d info\n\n",
		red(""), r.Summary.Critical, reset(),
		yellow(""), r.Summary.High, reset(),
		r.Summary.Medium, r.Summary.Low, r.Summary.Info,
	)

	for _, f := range r.Findings {
		icon := severityIcon(f.Severity)
		fmt.Printf("  %s [%s] %s\n", icon, f.Resource, f.Message)
		if f.Remediation != "" {
			fmt.Printf("       → %s\n", f.Remediation)
		}
	}
	fmt.Println()
}

func severityIcon(s report.Severity) string {
	switch s {
	case report.SeverityCritical:
		return red("✖ CRITICAL")
	case report.SeverityHigh:
		return red("✖ HIGH    ")
	case report.SeverityMedium:
		return yellow("⚠ MEDIUM  ")
	case report.SeverityLow:
		return "· LOW     "
	default:
		return "ℹ INFO    "
	}
}

func colorForGrade(g string) string {
	switch g {
	case "A":
		return green("")
	case "B":
		return green("")
	case "C":
		return yellow("")
	default:
		return red("")
	}
}

func bold(s string) string   { return "\033[1m" + s + "\033[0m" }
func red(s string) string    { return "\033[31m" + s }
func yellow(s string) string { return "\033[33m" + s }
func green(s string) string  { return "\033[32m" + s }
func reset() string          { return "\033[0m" }

func printHelp() {
	fmt.Print(`
Usage: kestrel <command> [flags]

Commands:
  scan <file|->   Scan a Kubernetes manifest or Helm chart
                  Use - to read from stdin
  version         Print version

Flags:
  --json          Output results as JSON
  --help, -h      Show this help

Examples:
  kestrel scan ./manifests/deployment.yaml
  helm template ./my-chart | kestrel scan -
  kestrel scan ./chart --json | jq .score
`)
}
