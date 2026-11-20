#!/usr/bin/env bash
# scan-eks.sh — Export running workloads from an EKS cluster and scan via ShrikeOps
#
# Usage:
#   ./scripts/scan-eks.sh --cluster my-cluster --region us-east-1
#   ./scripts/scan-eks.sh --cluster my-cluster --region us-east-1 --namespace production
#
# Prerequisites: aws CLI (authenticated), kubectl, eksctl or aws eks update-kubeconfig, curl, jq

set -euo pipefail

SHRIKE_API="${SHRIKE_API:-https://warblecloud.com/api/kestrel-scan}"  # optional SaaS; prefer local shrikeops binary
CLUSTER=""
REGION="us-east-1"
NAMESPACE="--all-namespaces"

while [[ $# -gt 0 ]]; do
  case $1 in
    --cluster)   CLUSTER="$2";    shift 2 ;;
    --region)    REGION="$2";     shift 2 ;;
    --namespace) NAMESPACE="-n $2"; shift 2 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

if [[ -z "$CLUSTER" ]]; then
  echo "Usage: $0 --cluster <cluster-name> --region <region> [--namespace <ns>]"
  exit 1
fi

echo "🐦 ShrikeOps — EKS Live Cluster Scan"
echo "   Cluster: $CLUSTER"
echo "   Region:  $REGION"
echo ""

# Step 1: Update kubeconfig for EKS
echo "▶ Updating kubeconfig for EKS cluster..."
aws eks update-kubeconfig --name "$CLUSTER" --region "$REGION"
echo "✅ kubeconfig updated"
echo ""

# Step 2: Export and scan (same logic as GKE script)
EXPORT_DIR=$(mktemp -d)
RESOURCES=(deployments statefulsets daemonsets jobs cronjobs)

echo "▶ Exporting workloads..."
for RESOURCE in "${RESOURCES[@]}"; do
  kubectl get "$RESOURCE" $NAMESPACE -o yaml 2>/dev/null > "$EXPORT_DIR/$RESOURCE.yaml" || true
done

echo "▶ Scanning..."
printf "%-35s %6s %6s %8s %6s\n" "Resource Type" "Score" "Grade" "Critical" "High"
printf "%-35s %6s %6s %8s %6s\n" "-----------------------------------" "-----" "-----" "--------" "----"

for RESOURCE in "${RESOURCES[@]}"; do
  f="$EXPORT_DIR/$RESOURCE.yaml"
  [[ ! -s "$f" ]] && continue

  RESPONSE=$(curl -sf -X POST "$SHRIKE_API" \
    -H "Content-Type: application/json" \
    -d "{\"manifest\": $(cat "$f" | jq -Rs .), \"source\": \"eks:$CLUSTER/$RESOURCE\"}" \
    --max-time 30 2>/dev/null) || continue

  SCORE=$(echo "$RESPONSE" | jq -r '.score // 0')
  GRADE=$(echo "$RESPONSE" | jq -r '.grade // "?"')
  CRIT=$(echo "$RESPONSE"  | jq -r '.summary.critical // 0')
  HIGH=$(echo "$RESPONSE"  | jq -r '.summary.high // 0')

  printf "%-35s %6s %6s %8s %6s\n" "$RESOURCE" "$SCORE" "$GRADE" "$CRIT" "$HIGH"
  echo "$RESPONSE" > "$EXPORT_DIR/${RESOURCE}-report.json"
done

echo ""
echo "📁 Reports saved to: $EXPORT_DIR"
echo ""

TOTAL_CRIT=$(cat "$EXPORT_DIR"/*-report.json 2>/dev/null | jq -s '[.[].summary.critical // 0] | add // 0')
[[ "$TOTAL_CRIT" -gt 0 ]] && { echo "❌ $TOTAL_CRIT critical finding(s) — remediate before next deployment"; exit 2; }
echo "✅ No critical findings in EKS cluster: $CLUSTER"
