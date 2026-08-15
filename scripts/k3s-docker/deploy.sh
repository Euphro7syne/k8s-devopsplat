#!/usr/bin/env bash
set -euo pipefail

PRESERVE_EXISTING_SECRET="${PRESERVE_EXISTING_SECRET:-false}"

kubectl apply -f deploy/k3s/namespace.yaml
if [[ "${PRESERVE_EXISTING_SECRET}" == "true" ]]; then
  if ! kubectl -n ops-platform get secret ops-server-secrets >/dev/null 2>&1; then
    echo "PRESERVE_EXISTING_SECRET=true but Secret/ops-server-secrets does not exist" >&2
    exit 1
  fi
  echo "Preserving existing Secret/ops-server-secrets"
else
  kubectl apply -f deploy/k3s/server-secret.yaml
fi
kubectl apply -f deploy/k3s/postgres.yaml
kubectl -n ops-platform rollout status statefulset/ops-postgres --timeout=180s

kubectl apply -f deploy/k3s/rbac.yaml
kubectl apply -f deploy/k3s/server-config.yaml
kubectl apply -f deploy/k3s/server.yaml
kubectl apply -f deploy/k3s/web.yaml
kubectl apply -f deploy/k3s/ingress.yaml
kubectl -n ops-platform rollout restart deployment/ops-server deployment/ops-web
kubectl -n ops-platform rollout status deploy/ops-server --timeout=120s
kubectl -n ops-platform rollout status deploy/ops-web --timeout=120s
kubectl -n ops-platform get pods,pvc,service -o wide
