package report

// Severity levels for scan findings
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Category of finding
type Category string

const (
	CategoryHelm        Category = "helm_lint"
	CategoryYAML        Category = "yaml_lint"
	CategoryDeprecated  Category = "deprecated_api"
	CategorySecurity    Category = "security_policy"
	CategoryReliability Category = "reliability"
	CategoryVuln        Category = "vulnerability"
	CategoryScanner     Category = "scanner" // scanner health (missing / failing tools)
)

// GradeIncomplete is the grade reported when one or more checks could not run.
const GradeIncomplete = "N/A"

// Finding is a single scan result item
type Finding struct {
	ID          string   `json:"id"`
	Category    Category `json:"category"`
	Severity    Severity `json:"severity"`
	Resource    string   `json:"resource"` // e.g. "Deployment/my-app"
	Message     string   `json:"message"`
	Remediation string   `json:"remediation"`
	DocsURL     string   `json:"docs_url,omitempty"`
	CVE         string   `json:"cve,omitempty"`
	CVSSScore   float64  `json:"cvss_score,omitempty"`
}

// SkippedCheck describes a check that could not run (tool missing or failed).
// The category it feeds is not scored and the scan is marked incomplete.
type SkippedCheck struct {
	Tool     string   `json:"tool"`
	Category Category `json:"category"`
	Reason   string   `json:"reason"`
	Install  string   `json:"install,omitempty"`
}

// ScoreBreakdown holds the weighted score components.
//
// When Complete is false, at least one check was skipped (see SkippedChecks):
// the skipped categories are reported as 0 and excluded from Overall, which is
// re-weighted over the categories that did run, and Grade is "N/A".
type ScoreBreakdown struct {
	Security      float64        `json:"security"`      // Polaris security checks (weight 40%)
	APIStability  float64        `json:"api_stability"` // Pluto (weight 20%)
	Reliability   float64        `json:"reliability"`   // kube-score (weight 25%)
	LintClean     float64        `json:"lint_clean"`    // helm+yaml (weight 15%)
	Overall       float64        `json:"overall"`
	Grade         string         `json:"grade"` // A/B/C/D/F, or N/A when incomplete
	Complete      bool           `json:"complete"`
	SkippedChecks []SkippedCheck `json:"skipped_checks,omitempty"`
}

// ScanReport is the full output of a ShrikeOps Scan
type ScanReport struct {
	ScanID    string         `json:"scan_id"`
	CreatedAt string         `json:"created_at"`
	Source    string         `json:"source"` // filename or repo URL
	Score     ScoreBreakdown `json:"score"`
	Findings  []Finding      `json:"findings"`
	Summary   ScanSummary    `json:"summary"`
}

// ScanSummary counts findings by severity
type ScanSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Info     int `json:"info"`
	Total    int `json:"total"`
	Passed   int `json:"passed"`
}
