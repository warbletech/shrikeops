# ShrikeOps 🐦

> The Kubernetes security scanner that catches bugs and impales them on thorns.

**ShrikeOps** is a self-hostable, open-source Kubernetes manifest & Helm chart analyser built by [Warble Cloud](https://warblecloud.com). Like the shrike — a carnivorous songbird that hunts, catches, and impales its prey — ShrikeOps hunts misconfigurations, catches vulnerabilities, and pins them to a scored report before they reach production.

Part of the **Warble Cloud** platform · [warblecloud.com/shrikeops](https://warblecloud.com/kestrel)

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

## Quick Start

### Run the scanner locally

```bash
# Install scan tools
brew install helm kube-score
# Install pluto
curl -sSL https://github.com/FairwindsOps/pluto/releases/latest/download/pluto_darwin_amd64.tar.gz | tar xz -C /usr/local/bin
# Install polaris
curl -sSL https://github.com/FairwindsOps/polaris/releases/latest/download/polaris_darwin_amd64.tar.gz | tar xz -C /usr/local/bin

cd scanner
go run ./cmd/main.go
```

### Scan a manifest

```bash
curl -X POST http://localhost:8090/scan \
  -H "Content-Type: application/json" \
  -d '{"source": "my-app.yaml", "manifest": "<paste YAML here>"}'
```

### Docker

```bash
docker build -t shrikeops-scanner ./scanner
docker run -p 8090:8090 shrikeops-scanner
```

---

## Roadmap

- [ ] CLI tool: `shrikeops scan ./chart`
- [ ] GitHub App — scan PRs automatically
- [ ] Helm chart upload via UI
- [ ] Git webhook integration
- [ ] Multi-cluster score history
- [ ] Slack / PagerDuty alerts on score drop

---

## License

Apache-2.0 © Warble Cloud / ChirpStack LLP
