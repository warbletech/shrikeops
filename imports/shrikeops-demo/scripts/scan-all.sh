#!/usr/bin/env bash
# scan-all.sh — Scan every YAML in the repo and print a summary table
# Usage: ./scripts/scan-all.sh [directory] [--fail-on-critical]
#
# Examples:
#   ./scripts/scan-all.sh                        # scan k8s/
#   ./scripts/scan-all.sh k8s/good               # scan only good configs
#   ./scripts/scan-all.sh k8s/ --fail-on-critical

set -euo pipefail

SHRIKE_API="${SHRIKE_API:-https://warblecloud.com/api/kestrel-scan}"  # optional SaaS; prefer local shrikeops binary
DIR="${1:-k8s}"
FAIL_ON_CRITICAL=false
[[ "${2:-}" == "--fail-on-critical" ]] && FAIL_ON_CRITICAL=true

echo "🐦 ShrikeOps — scanning all YAMLs in: $DIR"
echo ""
printf "%-45s %6s %6s %8s %6s %6s\n" "File" "Score" "Grade" "Critical" "High" "Status"
printf "%-45s %6s %6s %8s %6s %6s\n" "---------------------------------------------" "-----" "-----" "--------" "----" "------"

OVERALL_FAIL=false

find "$DIR" -name "*.yaml" -o -name "*.yml" | sort | while read -r f; do
  MANIFEST=$(cat "$f")
  RESPONSE=$(curl -sf -X POST "$SHRIKE_API" \
    -H "Content-Type: application/json" \
    -d "{\"manifest\": $(echo "$MANIFEST" | jq -Rs .), \"source\": \"$f\"}" \
    --max-time 30 2>/dev/null) || { printf "%-45s %6s\n" "$f" "ERROR"; continue; }

  SCORE=$(echo "$RESPONSE" | jq -r '.score // 0')
  GRADE=$(echo "$RESPONSE" | jq -r '.grade // "?"')
  CRIT=$(echo "$RESPONSE"  | jq -r '.summary.critical // 0')
  HIGH=$(echo "$RESPONSE"  | jq -r '.summary.high // 0')

  if [[ "$CRIT" -gt 0 ]]; then
    STATUS="❌ FAIL"
    [[ "$FAIL_ON_CRITICAL" == "true" ]] && OVERALL_FAIL=true
  else
    STATUS="✅ PASS"
  fi

  printf "%-45s %6s %6s %8s %6s  %s\n" "$f" "$SCORE" "$GRADE" "$CRIT" "$HIGH" "$STATUS"
done

echo ""
if [[ "$OVERALL_FAIL" == "true" ]]; then
  echo "❌ Scan complete — critical findings detected. Fix before deploying."
  exit 1
else
  echo "✅ Scan complete"
fi
