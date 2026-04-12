package vulndb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const osvAPIBase = "https://api.osv.dev/v1"

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Package identifies a software package for vulnerability lookup
type Package struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"` // e.g. "Go", "npm", "PyPI", "Kubernetes"
	Version   string `json:"version,omitempty"`
}

// OSVVuln is a simplified OSV vulnerability record
type OSVVuln struct {
	ID       string    `json:"id"`
	Summary  string    `json:"summary"`
	Details  string    `json:"details"`
	Severity []Severity `json:"severity"`
	Modified string    `json:"modified"`
	Published string   `json:"published"`
	References []Ref   `json:"references"`
}

type Severity struct {
	Type  string `json:"type"`
	Score string `json:"score"` // CVSS v3 vector or score
}

type Ref struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type osvQueryRequest struct {
	Version string  `json:"version,omitempty"`
	Package Package `json:"package"`
}

type osvQueryResponse struct {
	Vulns []OSVVuln `json:"vulns"`
}

// QueryPackage queries OSV.dev for vulnerabilities affecting a given package+version
func QueryPackage(pkg Package) ([]OSVVuln, error) {
	payload := osvQueryRequest{
		Version: pkg.Version,
		Package: pkg,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	resp, err := httpClient.Post(
		fmt.Sprintf("%s/query", osvAPIBase),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("osv query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("osv returned %d: %s", resp.StatusCode, string(b))
	}

	var result osvQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return result.Vulns, nil
}

// QueryBatch queries OSV.dev for multiple packages in one request
func QueryBatch(packages []Package) (map[string][]OSVVuln, error) {
	type batchQuery struct {
		Version string  `json:"version,omitempty"`
		Package Package `json:"package"`
	}
	type batchRequest struct {
		Queries []batchQuery `json:"queries"`
	}
	type batchResult struct {
		Results []osvQueryResponse `json:"results"`
	}

	queries := make([]batchQuery, len(packages))
	for i, p := range packages {
		queries[i] = batchQuery{Version: p.Version, Package: p}
	}

	body, _ := json.Marshal(batchRequest{Queries: queries})
	resp, err := httpClient.Post(
		fmt.Sprintf("%s/querybatch", osvAPIBase),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("osv querybatch: %w", err)
	}
	defer resp.Body.Close()

	var result batchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode batch: %w", err)
	}

	out := make(map[string][]OSVVuln)
	for i, r := range result.Results {
		if i < len(packages) {
			key := fmt.Sprintf("%s@%s", packages[i].Name, packages[i].Version)
			out[key] = r.Vulns
		}
	}
	return out, nil
}
