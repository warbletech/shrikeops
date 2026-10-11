# Test fixtures

- `manifests/good`, `manifests/bad`: copied from
  [warbletech/shrikeops-demo](https://github.com/warbletech/shrikeops-demo) `k8s/` (Apache-2.0).
- `tools/<tool>/<dir>-<name>.json`: real output of the external tools for each manifest,
  used to test the parsers and scoring without the binaries installed.
  Captured with pluto v5.24.4, polaris 10.2.5, kube-score 1.20.0:

```bash
pluto detect - --output json --ignore-deprecations=false < m.yaml
polaris audit --audit-path m.yaml --format json   # AuditTime removed
kube-score score - --output-format json < m.yaml
```

Regenerate after upgrading a tool and re-run `go test ./...`.
