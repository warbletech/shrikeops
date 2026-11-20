# ShrikeOps — Connect to EKS and Scan a Live Cluster

## Prerequisites

- `aws` CLI installed and authenticated (`aws configure`)
- `kubectl` installed
- `curl` and `jq` installed

---

## Step 1 — Update kubeconfig for EKS

```bash
# Standard update
aws eks update-kubeconfig \
  --name MY_CLUSTER \
  --region us-east-1

# With a specific IAM role (recommended for CI)
aws eks update-kubeconfig \
  --name MY_CLUSTER \
  --region us-east-1 \
  --role-arn arn:aws:iam::123456789012:role/eks-admin-role
```

Verify:
```bash
kubectl get nodes
kubectl cluster-info
```

---

## Step 2 — Export and Scan Workloads

```bash
# Quick method — export all and scan
./scripts/scan-eks.sh --cluster MY_CLUSTER --region us-east-1

# Specific namespace
./scripts/scan-eks.sh --cluster MY_CLUSTER --region us-east-1 --namespace production
```

Manual approach:
```bash
kubectl get deployments --all-namespaces -o yaml > eks-deployments.yaml
./scripts/scan.sh eks-deployments.yaml
```

---

## Step 3 — Automate in CI/CD (GitHub Actions)

```yaml
- name: Scan EKS cluster manifests
  env:
    AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
    AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
  run: |
    aws eks update-kubeconfig --name ${{ vars.EKS_CLUSTER }} --region us-east-1
    kubectl get deployments --all-namespaces -o yaml > /tmp/eks.yaml
    
    RESPONSE=$(curl -sf -X POST https://warblecloud.com/api/shrikeops-scan \
      -H "Content-Type: application/json" \
      -d "{\"manifest\": $(cat /tmp/eks.yaml | jq -Rs .), \"source\": \"eks:${{ vars.EKS_CLUSTER }}\"}")
    
    echo "$RESPONSE" | jq .
    CRIT=$(echo "$RESPONSE" | jq -r '.summary.critical')
    [ "$CRIT" -gt 0 ] && exit 1
```

---

## IRSA (IAM Roles for Service Accounts) for Secure Access

```bash
# Create OIDC provider for the cluster
eksctl utils associate-iam-oidc-provider \
  --cluster MY_CLUSTER \
  --region us-east-1 \
  --approve

# Create service account with IAM role
eksctl create iamserviceaccount \
  --name shrike-scanner \
  --namespace ShrikeOps \
  --cluster MY_CLUSTER \
  --attach-policy-arn arn:aws:iam::aws:policy/AmazonEKSReadOnlyPolicy \
  --approve
```
