# ShrikeOps 🐦

> The Kubernetes security scanner that catches bugs and impales them on thorns.

**ShrikeOps** is a self-hostable, open-source Kubernetes manifest & Helm chart analyser built by [Warble Cloud](https://warblecloud.com). Like the shrike — a carnivorous songbird that hunts, catches, and impales its prey — ShrikeOps hunts misconfigurations, catches vulnerabilities, and pins them to a scored report before they reach production.

[![Release](https://img.shields.io/github/v/release/warbletech/shrikeops)](https://github.com/warbletech/shrikeops/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/warbletech/shrikeops)](https://goreportcard.com/report/github.com/warbletech/shrikeops)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

Part of the **Warble Cloud** platform · [warblecloud.com](https://warblecloud.com)

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

## Installation

### macOS & Linux (Homebrew)

```bash
# Add WarbleTech tap
brew tap warbletech/tap

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
docker pull ghcr.io/warbletech/shrikeops:latest
```

---

## Quick Start

### CLI Usage

```bash
# Scan a Kubernetes manifest file
shrikeops scan ./manifests/deployment.yaml

# Scan a Helm chart
shrikeops scan ./charts/my-app/

# Scan with output format options
shrikeops scan ./manifests/ -o json
shrikeops scan ./manifests/ -o table
```

### Run the Scanner Server Locally

```bash
# Install dependencies (if not using pre-built binary)
brew install helm kube-score
curl -sSL https://github.com/FairwindsOps/pluto/releases/latest/download/pluto_darwin_amd64.tar.gz | tar xz -C /usr/local/bin
curl -sSL https://github.com/FairwindsOps/polaris/releases/latest/download/polaris_darwin_amd64.tar.gz | tar xz -C /usr/local/bin

cd scanner
go run ./cmd/cli/main.go server
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
docker run -p 8090:8090 ghcr.io/warbletech/shrikeops:latest server
```

---

## Roadmap

- [x] CLI tool: `shrikeops scan ./chart` ✓ Released v0.1.0
- [x] Homebrew tap installation ✓ Available via `brew tap warbletech/tap`
- [x] GitHub Container Registry publishing ✓ `ghcr.io/warbletech/shrikeops`
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
