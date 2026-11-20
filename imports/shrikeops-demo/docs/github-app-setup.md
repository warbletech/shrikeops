# ShrikeOps GitHub App — PR Check Integration

Every PR that touches a `.yaml` or `.yml` file automatically triggers a ShrikeOps Scan. Results appear as a **GitHub Check** directly in the PR, with inline annotations per finding.

---

## How It Works

```
PR opened/updated
       │
       ▼
GitHub webhook ──► https://warblecloud.com/api/shrikeops-webhook
       │
       ▼
ShrikeOps fetches changed YAML files from PR diff
       │
       ▼
Scans each file: Polaris + kube-score + Pluto + OPA
       │
       ▼
Posts GitHub Check with:
  • Overall score and grade
  • Annotations per finding (line-level)
  • PASS/FAIL status (blocks merge if critical found)
```

---

## Setup Steps

### 1. Install the ShrikeOps GitHub App

Go to: **[github.com/apps/ShrikeOps](https://github.com/apps/ShrikeOps)**  
Click **Install** → select the repositories you want ShrikeOps to guard.

### 2. Configure Branch Protection

1. Go to repo → **Settings → Branches → Branch protection rules**
2. For `main` (or your default branch):
   - ✅ **Require status checks to pass before merging**
   - Add status check: `ShrikeOps Security Scan`
   - ✅ **Require branches to be up to date**

### 3. Verify a PR Check

Open any PR that modifies a `.yaml` file. Within seconds you'll see:

```
✅ ShrikeOps Security Scan — score: 91 (A)  [Details]
   or
❌ ShrikeOps Security Scan — 2 critical findings  [Details]
```

Clicking **Details** shows the full scored report with remediation guidance.

---

## GitHub Actions Workflow (Alternative)

If you prefer Actions over the GitHub App, use the included workflow:

```bash
# Already included at:
.github/workflows/ShrikeOps-scan.yml
```

This workflow:
- Runs on every push/PR touching YAML files
- Scans all files in `k8s/`
- Renders Helm charts and scans output
- Enforces a minimum score of 80 on `k8s/good/`
- Publishes results to the GitHub Actions Step Summary

---

## Interpreting Results

| Grade | Score | Meaning |
|-------|-------|---------|
| A+    | 95-100 | Excellent — production hardened |
| A     | 85-94  | Good — minor improvements possible |
| B     | 70-84  | Acceptable — high findings present |
| C     | 55-69  | Warning — review before deploy |
| D     | 40-54  | Poor — significant misconfigs |
| F     | 0-39   | Critical — do not deploy |

---

## Webhook Endpoint

```
POST https://warblecloud.com/api/shrikeops-webhook
Content-Type: application/json
X-Hub-Signature-256: sha256=<hmac-signature>
```

Only `pull_request` events are processed. All other event types return `200 OK` and are ignored.

---

## Support

- **Scanner**: [warblecloud.com/kestrel](https://warbleoss.org)
- **GitHub setup**: [warblecloud.com/kestrel/github](https://github.com/warbletech/shrikeops)
- **Email**: [contact@warblecloud.com](mailto:contact@warblecloud.com)
