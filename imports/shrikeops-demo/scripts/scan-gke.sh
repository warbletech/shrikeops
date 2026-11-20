#!/usr/bin/env bash
# scan-gke.sh — Export running workloads from a GKE cluster and scan via ShrikeOps
#
# Usage:
#   ./scripts/scan-gke.sh --project my-project --cluster my-cluster --region us-central1
#   ./scripts/scan-gke.sh --project my-project --cluster my-cluster --zone us-central1-a --namespace production
#
# Prerequisites: gcloud (authenticated), kubectl, curl, jq

set -euo pipefail

SHRIKE_API="${SHRIKE_API:-https://warblecloud.com/api/kestrel-scan}"  # optional SaaS; prefer local shrikeops binary
PROJECT=""
CLUSTER=""
REGION=""
ZONE=""
NAMESPACE="--all-namespaces"
KUBECONFIG_FILE="${KUBECONFIG:-$HOME/.kube/config}"

# Parse args
while [[ $# -gt 0 ]]; do
  case $1 in
    --project)   PROJECT="$2";    shift 2 ;;
    --cluster)   CLUSTER="$2";    shift 2 ;;
    --region)    REGION="$2";     shift 2 ;;
    --zone)      ZONE="$2";       shift 2 ;;
    --namespace) NAMESPACE="-n $2"; shift 2 ;;
    --kubeconfig) KUBECONFIG_FILE="$2"; shift 2 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

if [[ -z "$PROJECT" || -z "$CLUSTER" ]]; then
  echo "Usage: $0 --project <project> --cluster <cluster> [--region <region>|--zone <zone>] [--namespace <ns>]"
  exit 1
fi

echo "🐦 ShrikeOps — GKE Live Cluster Scan"
echo "   Project: $PROJECT"
echo "   Cluster: $CLUSTER"
echo "   Namespace: ${NAMESPACE/--all-namespaces/ALL}"
echo ""

# Step 1: Get credentials
echo "▶ Fetching GKE credentials..."
if [[ -n "$REGION" ]]; then
  gcloud container clusters get-credentials "$CLUSTER" \
    --region "$REGION" --project "$PROJECT"
elif [[ -n "$ZONE" ]]; then
  gcloud container clusters get-credentials "$CLUSTER" \
    --zone "$ZONE" --project "$PROJECT"
else
  echo "Error: provide --region or --zone"
  exit 1
fi

echo "✅ kubeconfig updated: $KUBECONFIG_FILE"
echo ""

# Step 2: Export workloads
EXPORT_DIR=$(mktemp -d)
echo "▶ Exporting workloads to $EXPORT_DIR ..."

RESOURCES=(deployments statefulsets daemonsets jobs cronjobs)
for RESOURCE in "${RESOURCES[@]}"; do
  echo "  Exporting $RESOURCE..."
  kubectl get "$RESOURCE" $NAMESPACE \
    -o yaml 2>/dev/null > "$EXPORT_DIR/$RESOURCE.yaml" || true
done

echo ""
echo "▶ Scanning exported manifests..."
echo ""
printf "%-35s %6s %6s %8s %6s\n" "Resource Type" "Score" "Grade" "Critical" "High"
printf "%-35s %6s %6s %8s %6s\n" "-----------------------------------" "-----" "-----" "--------" "----"

for RESOURCE in "${RESOURCES[@]}"; do
  f="$EXPORT_DIR/$RESOURCE.yaml"
  [[ ! -s "$f" ]] && continue

  RESPONSE=$(curl -sf -X POST "$SHRIKE_API" \
    -H "Content-Type: application/json" \
    -d "{\"manifest\": $(cat "$f" | jq -Rs .), \"source\": \"gke:$CLUSTER/$RESOURCE\"}" \
    --max-time 30 2>/dev/null) || { echo "  Scan failed for $RESOURCE"; continue; }

  SCORE=$(echo "$RESPONSE" | jq -r '.score // 0')
  GRADE=$(echo "$RESPONSE" | jq -r '.grade // "?"')
  CRIT=$(echo "$RESPONSE"  | jq -r '.summary.critical // 0')
  HIGH=$(echo "$RESPONSE"  | jq -r '.summary.high // 0')

  printf "%-35s %6s %6s %8s %6s\n" "$RESOURCE" "$SCORE" "$GRADE" "$CRIT" "$HIGH"

  # Save individual JSON report
  echo "$RESPONSE" > "$EXPORT_DIR/${RESOURCE}-report.json"
done

echo ""
echo "📁 Reports saved to: $EXPORT_DIR"
echo "   Open the HTML viewer: ./scripts/report-viewer.sh $EXPORT_DIR/deployments-report.json"
echo ""

# Step 3: Check for criticals across all
TOTAL_CRIT=0
for f in "$EXPORT_DIR"/*-report.json; do
  [[ -f "$f" ]] || continue
  C=$(jq -r '.summary.critical // 0' "$f")
  TOTAL_CRIT=$((TOTAL_CRIT + C))
done

if [[ "$TOTAL_CRIT" -gt 0 ]]; then
  echo "❌ $TOTAL_CRIT critical finding(s) detected in live cluster: $CLUSTER"
  echo "   Review and remediate before next deployment."
  exit 2
else
  echo "✅ No critical findings in live cluster: $CLUSTER"
fi
