# ShrikeOps Demo

> Intentional good/bad Kubernetes manifests for **[ShrikeOps](https://github.com/warbletech/shrikeops)** — the scanner that catches bugs and impales them on thorns.

Part of the [Warble Army](https://github.com/warbletech/warble-army) flock · **[warbleoss.org](https://warbleoss.org)**

[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

---

## What this repo demonstrates

| Capability | Description |
|---|---|
| **Raw YAML scanning** | Deployment, Service, RBAC, NetworkPolicy fixtures |
| **Helm chart scanning** | `helm template` output piped to ShrikeOps |
| **Bad config detection** | Intentionally misconfigured manifests (root, privileged, deprecated APIs) |
| **CI pipeline** | GitHub Actions workflow to scan YAMLs on push/PR |
| **HTML report viewer** | Offline scan report rendered in browser |

Great for **`good-first-issue`** contributions: add fixtures, improve docs, fix scan scripts.

---

## Quick start (local CLI)

Build the ShrikeOps CLI from the [warble-army](https://github.com/warbletech/warble-army) `shrikeops` submodule, then scan a fixture in this repo. Full prerequisites, the binary path, and how this differs from the SaaS helper scripts are in [Scan with a Local Binary Only](#scan-with-a-local-binary-only).

```bash
git clone --recurse-submodules https://github.com/warbletech/warble-army.git
cd warble-army/shrikeops
make build
./bin/shrikeops scan ../shrikeops-demo/k8s/bad/root-container.yaml
```

`k8s/bad/root-container.yaml` is an intentionally misconfigured manifest checked into this repo. With `pluto`, `polaris`, and `kube-score` on `PATH`, a critical or high finding exits 2. That exit code comes from the local binary.

### Live MCP (readonly demo)

Pair findings with live cluster tools via Starling Lite:

```text
https://mcp.warbleoss.org
```

---

## Scan with a Local Binary Only

This workflow scans a manifest with a binary you build from the [warble-army](https://github.com/warbletech/warble-army) checkout. It does not require SaaS API keys, tokens, or secrets, and it does not use the SaaS API default.

The scanner is the ShrikeOps CLI in the `shrikeops` submodule. A warble-army checkout does not produce an executable named `warble-army`. The umbrella `Makefile` builds other tools (`bin/starling-lite`, `bin/heron-ops`). The scanner target is `make build` inside `shrikeops`, and it writes `shrikeops/bin/shrikeops`.

`shrikeops scan` has one mode. It lints YAML in-process and runs `pluto`, `polaris`, and `kube-score` when those binaries are on `PATH`. It does not read `SHRIKE_API` or any credential. Setting `SHRIKE_API` leaves the command unchanged.

The helper scripts in this repo are a separate path. `scripts/scan.sh`, `scripts/scan-all.sh`, `scripts/scan-helm.sh`, `scripts/scan-eks.sh`, and `scripts/scan-gke.sh` set `SHRIKE_API` to `https://warblecloud.com/api/kestrel-scan` when that variable is unset, then POST the manifest with `curl`. They do not execute `shrikeops`. `.github/workflows/shrikeops-scan.yml` posts to that same URL. Use the binary below when you want the local CLI.

`shrikeops/scanner/internal/vulndb` can query `https://api.osv.dev`. The `scan` command does not import that package, so this workflow does not call OSV and does not report OSV findings. `pluto`, `polaris`, and `kube-score` are separate programs. This path does not send the manifest to Warble Cloud. It is not an air-gapped guarantee for those other binaries.

### Prerequisites

- Git.
- Go 1.25 or newer. `shrikeops/scanner/go.mod` says `go 1.25.0`. The shrikeops CI workflow installs that version from the `go.mod` file.
- GNU make and a POSIX shell. The `build` recipe uses `mkdir -p` and `cd scanner && go build`.
- Network access on the first build, so the public Go module proxy can fetch the modules in `scanner/go.mod` and `scanner/go.sum`. That fetch does not use a Warble Cloud credential.
- For a complete graded scan, `pluto` 5.24.4, `polaris` 10.2.5, and `kube-score` 1.20.0 on `PATH`. Those versions are pinned in `shrikeops` `.github/workflows/scan.yml` and in the [shrikeops README](https://github.com/warbletech/shrikeops/blob/master/README.md).

Linux (amd64), as documented in the shrikeops README:

```bash
curl -fsSL https://github.com/FairwindsOps/pluto/releases/download/v5.24.4/pluto_5.24.4_linux_amd64.tar.gz | sudo tar -xz -C /usr/local/bin pluto
curl -fsSL https://github.com/FairwindsOps/polaris/releases/download/v10.2.5/polaris_10.2.5_linux_amd64.tar.gz | sudo tar -xz -C /usr/local/bin polaris
curl -fsSL https://github.com/zegl/kube-score/releases/download/v1.20.0/kube-score_1.20.0_linux_amd64.tar.gz | sudo tar -xz -C /usr/local/bin kube-score
```

On macOS the shrikeops README documents `brew install FairwindsOps/tap/pluto`, `brew install FairwindsOps/tap/polaris`, and `brew install kube-score`.

Without those three tools the CLI still scans locally. Their categories are skipped, the grade is `N/A`, stderr says `scan INCOMPLETE`, and the process exits 3 unless you pass `--allow-incomplete`.

### Build the binary

```bash
git clone --recurse-submodules https://github.com/warbletech/warble-army.git
cd warble-army
```

If `shrikeops/` is empty, the clone omitted submodules. From the warble-army root:

```bash
./scripts/submodules.sh init
```

`scripts/submodules.sh init` runs `git submodule sync --recursive` and then `git submodule update --init --recursive`. Then build:

```bash
cd shrikeops
make build
```

`shrikeops/Makefile` defines the scanner build as:

```makefile
VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)

build:
	mkdir -p bin
	cd scanner && go build -ldflags "$(LDFLAGS)" -o ../bin/shrikeops ./cmd/cli/main.go
```

Run `make build` from the `shrikeops` directory. The binary is `bin/shrikeops` inside that directory (`warble-army/shrikeops/bin/shrikeops`).

On Windows, PowerShell does not provide this `make`. Run `make build` from WSL, Git Bash, or another POSIX environment.

The warble-army README also shows `go run ./cmd/cli/main.go scan ...` from `shrikeops/scanner`. That compiles and runs the same CLI and does not leave `bin/shrikeops`. `make build` is the step that writes the binary.

### Run the binary

From the `shrikeops` submodule directory:

```bash
./bin/shrikeops version
./bin/shrikeops scan ../shrikeops-demo/k8s/bad/root-container.yaml
```

`version` prints `shrikeops version dev (commit <short-sha>)`. The short SHA is the `shrikeops` checkout, not this demo repo.

From a checkout of **this** repository, point at the binary you built and pass the fixture path:

```bash
/path/to/warble-army/shrikeops/bin/shrikeops scan k8s/bad/root-container.yaml
```

The example target `k8s/bad/root-container.yaml` is a local file in this repo. It is intentionally misconfigured. The scan does not need a cluster, kubeconfig, or cloud credentials. In a warble-army checkout the same file is `shrikeops-demo/k8s/bad/root-container.yaml` (the `shrikeops-demo` submodule).

Flags from `shrikeops scan` usage:

| Flag | Behavior |
|---|---|
| `--json` | JSON report on stdout. Warnings stay on stderr. |
| `--allow-incomplete` | Exit 0 when the scan is incomplete and there is no critical or high finding. Critical or high findings still exit 2. |
| `-` | Read the manifest from stdin. |

```bash
./bin/shrikeops scan ../shrikeops-demo/k8s/bad/root-container.yaml --json
```

Exit codes from the CLI: `0` when every check ran and there is no critical or high finding, `1` for usage or I/O errors, `2` for a critical or high finding, `3` when a required tool is missing or failed and `--allow-incomplete` was not set.

### Confirm the scan ran locally

- The process you started is `shrikeops/bin/shrikeops`.
- `./bin/shrikeops version` prints the commit baked in by `make build`.
- The report names the file (`root-container.yaml`) and does not print `API:` or `https://warblecloud.com`.
- With `pluto`, `polaris`, and `kube-score` on `PATH`, the scan of `k8s/bad/root-container.yaml` finishes with `"complete": true` in the JSON score and exits 2 because the fixture has high findings. Finding ids come from those local tools (`POL-…` from polaris, `KS-…` from kube-score).
- With those tools absent, stderr contains `shrikeops: WARNING: scan INCOMPLETE — skipped: pluto (not found on PATH)` and the exit code is 3. `--allow-incomplete` then exits 0, and the report stays incomplete.
- `SHRIKE_API`, API keys, and tokens are not arguments to this command. The helper scripts are the commands that print `API: $SHRIKE_API`.

### Troubleshooting

| What you see | What it means | What to do |
|---|---|---|
| `go: command not found`, or a `go.mod` version error | Go is missing or older than 1.25 | Install Go 1.25 or newer and run `make build` again |
| `make: command not found`, or the Makefile fails in PowerShell | `make build` is a POSIX Makefile | Run it in WSL, Git Bash, Linux, or macOS |
| `shrikeops/` is empty | Submodules were not checked out | From the warble-army root, `./scripts/submodules.sh init` |
| `go: downloading` fails during `make build` | Modules could not be fetched | The build needs outbound access to the public Go module proxy. No Warble API key is required |
| No file at `shrikeops/scanner/bin/shrikeops` | That is not the Makefile output | Run `shrikeops/bin/shrikeops` |
| `error reading` the yaml, exit 1 | The path is relative to the current directory | From `shrikeops/`, use `../shrikeops-demo/k8s/bad/root-container.yaml`. From this repo, use `k8s/bad/root-container.yaml` |
| Exit 3 and `scan INCOMPLETE` | `pluto`, `polaris`, or `kube-score` is missing or failed | Install the versions above, or rerun with `--allow-incomplete` when the checks that ran are enough |
| Exit 2 on `k8s/bad/root-container.yaml` | The local scan found a critical or high finding | Expected for this fixture once the external tools run |
| `scripts/scan.sh` prints `API: https://warblecloud.com/api/kestrel-scan` | That script is the SaaS helper | Invoke `shrikeops/bin/shrikeops scan` instead |
| `unknown command` | The word before the file was not `scan` | Use `shrikeops scan <file\|-> [--json] [--allow-incomplete]` |

---

## Repository layout

```
shrikeops-demo/
├── k8s/
│   ├── good/                   # Well-configured manifests
│   ├── bad/                    # Intentionally bad configs
│   └── helm/                   # Sample chart
├── scripts/                    # scan helpers
├── reports/                    # HTML report viewer
├── demo/                       # asciinema / static demo (if present)
└── docs/                       # GKE/EKS/GitHub App setup notes
```

---

## Docs

- [ShrikeOps](https://github.com/warbletech/shrikeops)
- [Heron Ops](https://github.com/warbletech/heron-ops)
- [Starling EX Community](https://github.com/warbletech/warble-army/blob/main/docs/STARLING_EX_CE.md)
- [warbleoss.org](https://warbleoss.org)

Optional commercial SaaS (not required for demos): [warblecloud.com](https://warblecloud.com)

---

## License

Apache-2.0 — see [LICENSE](LICENSE).

## Security

Report vulnerabilities to **security@warblecloud.com** — see [SECURITY.md](SECURITY.md).
