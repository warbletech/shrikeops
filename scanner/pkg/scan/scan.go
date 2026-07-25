// Package scan is the public API for embedding ShrikeOps scans (Starling Lite, CI).
package scan

import (
	"github.com/warbletech/shrikeops/scanner/internal/engine"
	"github.com/warbletech/shrikeops/scanner/internal/report"
)

// Finding is a single scan result.
type Finding = report.Finding

// ScanReport is the full output of a scan.
type ScanReport = report.ScanReport

// ScoreBreakdown is the weighted score components.
type ScoreBreakdown = report.ScoreBreakdown

// ScanSummary counts findings by severity.
type ScanSummary = report.ScanSummary

// Request is the input for a scan.
type Request struct {
	Manifest string // raw YAML content
	Source   string // filename or label
}

// Run executes all scan checks and returns a report.
func Run(req Request) (*ScanReport, error) {
	return engine.Run(engine.ScanRequest{
		Manifest: req.Manifest,
		Source:   req.Source,
	})
}
