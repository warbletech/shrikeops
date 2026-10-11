# ShrikeOps 🐦

<p align="center">
  <img src="https://raw.githubusercontent.com/warbletech/warble-army/main/brand/shrike-ops/shrike-ops-logo.jpg" alt="ShrikeOps" width="120" />
</p>

> The Kubernetes security scanner that catches bugs and impales them on thorns.

**ShrikeOps** is a self-hostable, open-source Kubernetes manifest & Helm chart analyser built by [Warble Cloud](https://warblecloud.com) / the [Warble Army](https://github.com/warbletech/warble-army) flock. Like the shrike — a carnivorous songbird that hunts, catches, and impales its prey — ShrikeOps hunts misconfigurations, catches vulnerabilities, and pins them to a scored report before they reach production.

Community Edition home (planned): **[warbleoss.org](https://warbleoss.org)** · Pairs with **Heron Ops** (cluster MCP) and **Starling Lite**.

[![Release](https://img.shields.io/github/v/release/warbletech/shrikeops)](https://github.com/warbletech/shrikeops/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/warbletech/shrikeops)](https://goreportcard.com/report/github.com/warbletech/shrikeops)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

Part of **Warble OSS** ([warbleoss.org](https://warbleoss.org)) and the Warble Cloud product family · [warblecloud.com](https://warblecloud.com)

---

## What it scans

| Tool | Checks | License |
|---|---|---|
| **YAML Lint** | Syntax, indentation, structure | built-in |
| **Fairwinds Pluto** | Deprecated & removed K8s APIs | Apache-2.0 |
| **Fairwinds Polaris** | 30+ security best practices | Apache-2.0 |
| **kube-score** | Reliability & security scoring | MIT |
| **OPA Rego Policies** | Custom org security policies | Apache-2.0 |
| **OSV.dev VulnDB** | CVEs for container dependencies | Apache-2.0 |

---

## ShrikeOps Score

```
Overall Score = Security (40%) + API Stability (20%) + Reliability (25%) + Lint (15%)
```

Grades: **A** ≥90 · **B** ≥75 · **C** ≥60 · **D** ≥45 · **F** <45

| Category | Source | Severity mapping |
|---|---|---|
| Security | Polaris checks in the `Security` category | `danger` → high, `warning` → low |
| API Stability | Pluto | removed API → high, deprecated API → medium |
| Reliability | kube-score | `CRITICAL` → medium, `WARNING` → low; **skipped checks are ignored** |
| Lint | built-in YAML parser | parse error → high |

Penalty per finding: critical 25 · high 15 · medium 7 · low 3 · info 0 (each category is clamped to 0–100).

### Incomplete scans

ShrikeOps never scores a check that did not run. If `pluto`, `polaris` or `kube-score` is missing from `PATH`
(or fails / returns unparseable output):

- the category it feeds is **not scored** (`n/a` in the CLI, `0` in JSON) and the overall score is re-weighted over the checks that did run;
- `score.complete` is `false`, `score.grade` is `"N/A"` and `score.skipped_checks` lists each tool, the reason and how to install it;
- the CLI prints a warning to stderr and **exits 3**, so a CI gate cannot silently pass. Use `--allow-incomplete` to exit 0 anyway (the report is still marked incomplete).

```json
"score": {
  "security": 0, "api_stability": 100, "reliability": 51, "lint_clean": 100,
  "overall": 79.6, "grade": "N/A", "complete": false,
  "skipped_checks": [{"tool": "polaris", "category": "security_policy", "reason": "not found on PATH", "install": "https://github.com/FairwindsOps/polaris/releases ..."}]
}
```

---

## Project Structure

```
shrikeops/
├── scanner/        # Go — scan engine (Pluto, Polaris, kube-score, YAML lint)
│   ├── cmd/        # main entrypoint + HTTP server (Gin)
│   └── internal/
│       ├── engine/ # scan orchestration + HTTP handler
│       ├── vulndb/ # OSV.dev client
│       └── report/ # shared types
├── api/            # (future) auth gateway, result storage, webhooks
├── frontend/       # Next.js dashboard + scan UI
├── policies/       # OPA Rego policy bundles
└── .github/        # CI workflows
```

---

## Requirements

The `shrikeops` binary orchestrates three external scanners. **All three must be on `PATH`** for a complete (graded) scan:

| Tool | Tested version | Install |
|---|---|---|
| [pluto](https://github.com/FairwindsOps/pluto) | 5.24.4 | `brew install FairwindsOps/tap/pluto` or [releases](https://github.com/FairwindsOps/pluto/releases) |
| [polaris](https://github.com/FairwindsOps/polaris) | 10.2.5 | `brew install FairwindsOps/tap/polaris` or [releases](https://github.com/FairwindsOps/polaris/releases) |
| [kube-score](https://github.com/zegl/kube-score) | 1.20.0 | `brew install kube-score` or [releases](https://github.com/zegl/kube-score/releases) |

Linux (amd64; use `arm64` on ARM):

```bash
curl -fsSL https://github.com/FairwindsOps/pluto/releases/download/v5.24.4/pluto_5.24.4_linux_amd64.tar.gz | sudo tar -xz -C /usr/local/bin pluto
curl -fsSL https://github.com/FairwindsOps/polaris/releases/download/v10.2.5/polaris_10.2.5_linux_amd64.tar.gz | sudo tar -xz -C /usr/local/bin polaris
curl -fsSL https://github.com/zegl/kube-score/releases/download/v1.20.0/kube-score_1.20.0_linux_amd64.tar.gz | sudo tar -xz -C /usr/local/bin kube-score
```

The Docker image (`ghcr.io/warble-tech/shrikeops`) already contains all three, plus the `shrikeops` CLI.
Without them, scans are reported as **incomplete** (see [Incomplete scans](#incomplete-scans)).

## Installation

### macOS & Linux (Homebrew)

```bash
# Add WarbleTech tap
brew tap warble-tech/tap

# Install ShrikeOps
brew install shrikeops

# Upgrade to latest version
brew upgrade shrikeops
```

### GitHub Releases (All Platforms)

Download pre-built binaries from [GitHub Releases](https://github.com/warbletech/shrikeops/releases):

| Platform | Binary |
|----------|--------|
| macOS (Intel) | `shrikeops-darwin-amd64` |
| macOS (Apple Silicon) | `shrikeops-darwin-arm64` |
| Linux (x86_64) | `shrikeops-linux-amd64` |
| Linux (ARM64) | `shrikeops-linux-arm64` |
| Windows (x86_64) | `shrikeops-windows-amd64.exe` |

```bash
# Example: macOS Intel
curl -L https://github.com/warbletech/shrikeops/releases/latest/download/shrikeops-darwin-amd64 -o shrikeops
chmod +x shrikeops
sudo mv shrikeops /usr/local/bin/
```

### Docker / GitHub Container Registry

```bash
docker pull ghcr.io/warble-tech/shrikeops:latest
```

---

## Quick Start

### CLI Usage

```bash
# Scan a Kubernetes manifest file (multi-document YAML is fine)
shrikeops scan ./manifests/deployment.yaml

# Scan a Helm chart: render it first, then scan from stdin
helm template ./charts/my-app | shrikeops scan -

# JSON output (stdout is pure JSON; the banner only goes to an interactive stderr)
shrikeops scan ./manifests/deployment.yaml --json | jq .score

# Run without the external tools (report is marked incomplete, exit 0)
shrikeops scan ./manifests/deployment.yaml --allow-incomplete
```

Exit codes: `0` ok · `1` usage/I/O error · `2` critical or high findings · `3` scan incomplete (missing/failed tool).

### Run the Scanner Server Locally

```bash
# Install dependencies (if not using pre-built binary)
# Install pluto, polaris and kube-score first (see Requirements)

cd scanner
go run ./cmd/main.go        # HTTP API on :8090 (POST /scan)
go test ./...             # unit tests (no external tools needed)
```

### API Server Usage

```bash
# Start server
curl -X POST http://localhost:8090/scan \
  -H "Content-Type: application/json" \
  -d '{"source": "my-app.yaml", "manifest": "<paste YAML here>"}'
```

### Docker

```bash
# HTTP scanner API on :8090
docker run -p 8090:8090 ghcr.io/warble-tech/shrikeops:latest
# CLI
docker run -i --rm --entrypoint shrikeops ghcr.io/warble-tech/shrikeops:latest scan - < deployment.yaml
```

---

## Roadmap

- [x] CLI tool: `shrikeops scan ./chart` ✓ Released v0.1.0
- [x] Homebrew tap installation ✓ Available via `brew tap warble-tech/tap`
- [x] GitHub Container Registry publishing ✓ `ghcr.io/warble-tech/shrikeops`
- [ ] GitHub App — scan PRs automatically
- [ ] Helm chart upload via UI
- [ ] Git webhook integration
- [ ] Multi-cluster score history dashboard
- [ ] Slack / PagerDuty alerts on score drop
- [ ] SARIF output format for GitHub Advanced Security
- [ ] Policy-as-code marketplace

---

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Apache-2.0 © Warble Cloud / ChirpStack LLP
