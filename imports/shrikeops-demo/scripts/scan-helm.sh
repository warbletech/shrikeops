#!/usr/bin/env bash
# scan-helm.sh — Render a Helm chart and scan the output via ShrikeOps
# Usage: ./scripts/scan-helm.sh <chart-dir> [release-name] [namespace] [values-file]
#
# Examples:
#   ./scripts/scan-helm.sh k8s/helm
#   ./scripts/scan-helm.sh k8s/helm my-release production
#   ./scripts/scan-helm.sh k8s/helm my-release production custom-values.yaml
#
# Prerequisites: helm, curl, jq

set -euo pipefail

SHRIKE_API="${SHRIKE_API:-https://warblecloud.com/api/kestrel-scan}"  # optional SaaS; prefer local shrikeops binary
CHART_DIR="${1:-k8s/helm}"
RELEASE="${2:-demo-release}"
NAMESPACE="${3:-production}"
VALUES="${4:-}"

if ! command -v helm &>/dev/null; then
  echo "Error: helm is not installed. Install from https://helm.sh/docs/intro/install/"
  exit 1
fi

echo "🐦 ShrikeOps — Helm chart scan"
echo "   Chart:     $CHART_DIR"
echo "   Release:   $RELEASE"
echo "   Namespace: $NAMESPACE"
echo ""

# Build helm template command
HELM_CMD="helm template $RELEASE $CHART_DIR --namespace $NAMESPACE"
[[ -n "$VALUES" ]] && HELM_CMD="$HELM_CMD -f $VALUES"

echo "Rendering: $HELM_CMD"
RENDERED=$($HELM_CMD)
echo "Rendered ${#RENDERED} bytes of YAML — scanning..."
echo ""

RESPONSE=$(curl -sf -X POST "$SHRIKE_API" \
  -H "Content-Type: application/json" \
  -d "{\"manifest\": $(echo "$RENDERED" | jq -Rs .), \"source\": \"helm:$CHART_DIR\"}" \
  --max-time 30)

SCORE=$(echo "$RESPONSE" | jq -r '.score')
GRADE=$(echo "$RESPONSE" | jq -r '.grade')
CRIT=$(echo "$RESPONSE"  | jq -r '.summary.critical')
HIGH=$(echo "$RESPONSE"  | jq -r '.summary.high')
MED=$(echo "$RESPONSE"   | jq -r '.summary.medium')
SCAN_ID=$(echo "$RESPONSE" | jq -r '.scan_id')

echo "╔══════════════════════════════════════════════╗"
printf "║  Helm Chart: %-32s ║\n" "$CHART_DIR"
printf "║  Score: %-3s  Grade: %-2s  ID: %-16s  ║\n" "$SCORE" "$GRADE" "$SCAN_ID"
echo "╠══════════════════════════════════════════════╣"
printf "║  🔴 Critical: %-3s  🟠 High: %-3s  🟡 Med: %-3s  ║\n" "$CRIT" "$HIGH" "$MED"
echo "╚══════════════════════════════════════════════╝"
echo ""

# Print all findings grouped by severity
for SEV in CRITICAL HIGH MEDIUM LOW; do
  COUNT=$(echo "$RESPONSE" | jq -r "[.findings[] | select(.severity == \"$SEV\")] | length")
  if [[ "$COUNT" -gt 0 ]]; then
    echo "── $SEV ($COUNT) ──────────────────────────────"
    echo "$RESPONSE" | jq -r \
      ".findings[] | select(.severity == \"$SEV\") | \"  [\(.tool)] \(.rule)\n  Resource: \(.resource // \"n/a\")\n  \(.message)\n  Fix: \(.remediation // \"See ShrikeOps docs\")\n\""
  fi
done

[[ "$CRIT" -gt 0 ]] && { echo "❌ FAILED — critical findings"; exit 2; }
echo "✅ Helm scan complete — score: $SCORE"
