# ShrikeOps — Connect to GKE and Scan a Live Cluster

## Prerequisites

- `gcloud` CLI installed and authenticated (`gcloud auth login`)
- `kubectl` installed
- `curl` and `jq` installed

---

## Step 1 — Get GKE Credentials

```bash
# Standard regional cluster
gcloud container clusters get-credentials MY_CLUSTER \
  --region us-central1 \
  --project MY_PROJECT

# Zonal cluster
gcloud container clusters get-credentials MY_CLUSTER \
  --zone us-central1-a \
  --project MY_PROJECT
```

Verify connection:
```bash
kubectl get nodes
kubectl cluster-info
```

---

## Step 2 — Export Running Workloads

```bash
# Export all Deployments across all namespaces
kubectl get deployments --all-namespaces -o yaml > deployments.yaml

# Export StatefulSets, DaemonSets, Jobs
kubectl get statefulsets --all-namespaces -o yaml > statefulsets.yaml
kubectl get daemonsets   --all-namespaces -o yaml > daemonsets.yaml

# Export from a specific namespace
kubectl get deployments -n production -o yaml > production-deployments.yaml
```

---

## Step 3 — Scan with ShrikeOps

```bash
# Scan a single exported file
./scripts/scan.sh deployments.yaml

# Scan everything in one go using the GKE script
./scripts/scan-gke.sh \
  --project MY_PROJECT \
  --cluster MY_CLUSTER \
  --region us-central1 \
  --namespace production
```

---

## Step 4 — View the Report

```bash
# Scan and save JSON report
curl -s -X POST https://warblecloud.com/api/shrikeops-scan \
  -H "Content-Type: application/json" \
  -d "{\"manifest\": $(cat deployments.yaml | jq -Rs .), \"source\": \"gke:MY_CLUSTER/deployments\"}" \
  > report.json

# Open in HTML viewer
./scripts/report-viewer.sh report.json
```

Or open `reports/viewer.html` in a browser and drag-drop `report.json`.

---

## Step 5 — Automate in CI/CD (Cloud Build)

Add this step to your `cloudbuild.yaml` to scan before deployment:

```yaml
- name: 'gcr.io/google.com/cloudsdktool/cloud-sdk'
  entrypoint: bash
  args:
    - '-c'
    - |
      kubectl get deployments --all-namespaces -o yaml > /tmp/workloads.yaml
      RESPONSE=$(curl -sf -X POST https://warblecloud.com/api/shrikeops-scan \
        -H "Content-Type: application/json" \
        -d "{\"manifest\": $(cat /tmp/workloads.yaml | jq -Rs .), \"source\": \"gke:$CLUSTER_NAME\"}")
      CRIT=$(echo "$RESPONSE" | jq -r '.summary.critical // 0')
      [ "$CRIT" -gt 0 ] && { echo "❌ Critical findings — blocking deploy"; exit 1; }
      echo "✅ ShrikeOps Scan passed"
```

---

## Workload Identity (Recommended for Production)

Instead of service account key files, use Workload Identity to let your CI/CD pipeline authenticate to GKE without any credentials file:

```bash
# Enable Workload Identity on the cluster
gcloud container clusters update MY_CLUSTER \
  --region us-central1 \
  --workload-pool=MY_PROJECT.svc.id.goog

# Bind a Kubernetes ServiceAccount to a GCP ServiceAccount
gcloud iam service-accounts add-iam-policy-binding \
  shrike-scanner@MY_PROJECT.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:MY_PROJECT.svc.id.goog[ShrikeOps/shrike-sa]"
```

---

## Useful kubectl Commands for Security Auditing

```bash
# Find containers running as root
kubectl get pods --all-namespaces -o json | \
  jq '.items[] | select(.spec.containers[].securityContext.runAsUser == 0) | .metadata.name'

# Find pods with no resource limits
kubectl get pods --all-namespaces -o json | \
  jq '.items[] | select(.spec.containers[].resources.limits == null) | .metadata.name'

# Find privileged containers
kubectl get pods --all-namespaces -o json | \
  jq '.items[] | select(.spec.containers[].securityContext.privileged == true) | .metadata.name'

# Find pods with hostNetwork
kubectl get pods --all-namespaces -o json | \
  jq '.items[] | select(.spec.hostNetwork == true) | .metadata.name'
```
