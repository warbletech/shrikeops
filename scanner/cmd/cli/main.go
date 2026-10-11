package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/warbletech/shrikeops/scanner/internal/engine"
	"github.com/warbletech/shrikeops/scanner/internal/report"
)

// Set via -ldflags "-X main.version=... -X main.commit=..." (see Makefile).
var (
	version = "0.1.0"
	commit  = "unknown"
)

// Exit codes.
const (
	exitOK         = 0
	exitUsage      = 1
	exitFindings   = 2 // critical or high findings
	exitIncomplete = 3 // one or more checks could not run (see --allow-incomplete)
)

func bannerText() string {
	return `
 ____  _          _ _         ___
/ ___|| |__  _ __(_) | _____ / _ \ _ __  ___
\___ \| '_ \| '__| | |/ / _ \ | | | '_ \/ __|
 ___) | | | | |  | |   <  __/ |_| | |_) \__ \
|____/|_| |_|_|  |_|_|\_\___|\___/| .__/|___/
  Kubernetes Helm Safety Gate     |_|  v` + version + `
`
}

// isTerminal reports whether f is an interactive terminal (no x/term dependency).
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// maybeBanner prints the banner to stderr, only for interactive human use.
func maybeBanner(jsonOutput bool) {
	if jsonOutput || !isTerminal(os.Stderr) || os.Getenv("SHRIKEOPS_NO_BANNER") != "" {
		return
	}
	fmt.Fprint(os.Stderr, bannerText())
}

type options struct {
	path            string
	json            bool
	allowIncomplete bool
}

func parseScanArgs(args []string) (options, error) {
	var o options
	for _, a := range args {
		switch a {
		case "--json", "-o=json":
			o.json = true
		case "--allow-incomplete":
			o.allowIncomplete = true
		default:
			if o.path != "" {
				return o, fmt.Errorf("unexpected argument %q", a)
			}
			o.path = a
		}
	}
	if o.path == "" {
		return o, fmt.Errorf("missing <file|->")
	}
	return o, nil
}

// exitCode decides the process exit status for a finished scan.
func exitCode(r *report.ScanReport, allowIncomplete bool) int {
	if r.Summary.Critical > 0 || r.Summary.High > 0 {
		return exitFindings
	}
	if !r.Score.Complete && !allowIncomplete {
		return exitIncomplete
	}
	return exitOK
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		maybeBanner(false)
		printHelp(stdout)
		return exitOK
	}
	switch args[0] {
	case "version", "--version":
		fmt.Fprintf(stdout, "shrikeops version %s (commit %s)\n", version, commit)
		return exitOK
	case "scan":
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		printHelp(stderr)
		return exitUsage
	}

	o, err := parseScanArgs(args[1:])
	if err != nil {
		fmt.Fprintf(stderr, "usage: shrikeops scan <file|-> [--json] [--allow-incomplete]: %v\n", err)
		return exitUsage
	}
	maybeBanner(o.json)

	var content []byte
	path := o.path
	if path == "-" {
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(stdin); err != nil {
			fmt.Fprintf(stderr, "error reading stdin: %v\n", err)
			return exitUsage
		}
		content = buf.Bytes()
		path = "stdin"
	} else {
		content, err = os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "error reading %s: %v\n", path, err)
			return exitUsage
		}
	}

	result, err := engine.Run(engine.ScanRequest{Manifest: string(content), Source: filepath.Base(path)})
	if err != nil {
		fmt.Fprintf(stderr, "scan error: %v\n", err)
		return exitUsage
	}

	if o.json {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(stderr, "encode: %v\n", err)
			return exitUsage
		}
	} else {
		printHuman(stdout, result)
	}
	if !result.Score.Complete {
		fmt.Fprintln(stderr, incompleteWarning(result, o.allowIncomplete))
	}
	return exitCode(result, o.allowIncomplete)
}

func incompleteWarning(r *report.ScanReport, allowed bool) string {
	msg := "shrikeops: WARNING: scan INCOMPLETE — skipped:"
	for _, s := range r.Score.SkippedChecks {
		msg += fmt.Sprintf(" %s (%s)", s.Tool, s.Reason)
	}
	msg += ". No grade assigned. Install the missing tools (see README#requirements)"
	if !allowed {
		msg += " or pass --allow-incomplete to exit 0."
	} else {
		msg += "."
	}
	return msg
}

func printHuman(w io.Writer, r *report.ScanReport) {
	s := r.Score
	fmt.Fprintf(w, "\n%s ShrikeOps Scan — %s%s\n", bold("📋"), r.Source, reset())
	fmt.Fprintf(w, "  Scan ID : %s\n", r.ScanID)
	if s.Complete {
		fmt.Fprintf(w, "  Score   : %s%.1f/100 (%s)%s\n", colorForGrade(s.Grade), s.Overall, s.Grade, reset())
	} else {
		fmt.Fprintf(w, "  Score   : %sINCOMPLETE — partial %.1f/100 over checks that ran, no grade%s\n", red(""), s.Overall, reset())
	}
	notRun := map[report.Category]bool{}
	for _, sk := range s.SkippedChecks {
		notRun[sk.Category] = true
	}
	cat := func(c report.Category, v float64) string {
		if notRun[c] {
			return "  n/a"
		}
		return fmt.Sprintf("%5.1f", v)
	}
	fmt.Fprintf(w, "  Security     : %s   API Stability: %s\n", cat(report.CategorySecurity, s.Security), cat(report.CategoryDeprecated, s.APIStability))
	fmt.Fprintf(w, "  Reliability  : %s   Lint Clean   : %s\n\n", cat(report.CategoryReliability, s.Reliability), cat(report.CategoryYAML, s.LintClean))

	for _, sk := range s.SkippedChecks {
		fmt.Fprintf(w, "  %s⚠ SKIPPED  %s: %s — install: %s%s\n", yellow(""), sk.Tool, sk.Reason, sk.Install, reset())
	}
	if len(s.SkippedChecks) > 0 {
		fmt.Fprintln(w)
	}

	if r.Summary.Total == 0 {
		fmt.Fprintln(w, green("  ✓ No issues found — chart looks healthy!")+reset())
		return
	}

	fmt.Fprintf(w, "  %s%d critical%s  %s%d high%s  %d medium  %d low  %d info\n\n",
		red(""), r.Summary.Critical, reset(),
		yellow(""), r.Summary.High, reset(),
		r.Summary.Medium, r.Summary.Low, r.Summary.Info,
	)

	for _, f := range r.Findings {
		if f.Category == report.CategoryScanner {
			continue // already shown as SKIPPED above
		}
		fmt.Fprintf(w, "  %s%s [%s] %s\n", severityIcon(f.Severity), reset(), f.Resource, f.Message)
		if f.Remediation != "" {
			fmt.Fprintf(w, "       → %s\n", f.Remediation)
		}
	}
	fmt.Fprintln(w)
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
	case "A", "B":
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

func printHelp(w io.Writer) {
	fmt.Fprint(w, `
usage: shrikeops <command> [flags]

Commands:
  scan <file|->   Scan a Kubernetes manifest (multi-doc YAML) or rendered Helm output
                  Use - to read from stdin
  version         Print version

Flags:
  --json              Output results as JSON (stdout is pure JSON)
  --allow-incomplete  Exit 0 even if pluto/polaris/kube-score are missing
  --help, -h          Show this help

Exit codes:
  0  no critical/high findings and every check ran
  1  usage or I/O error
  2  critical or high findings
  3  scan incomplete: a required tool (pluto, polaris, kube-score) is missing or failed

Examples:
  shrikeops scan ./manifests/deployment.yaml
  helm template ./my-chart | shrikeops scan -
  shrikeops scan ./deploy.yaml --json | jq .score
`)
}
