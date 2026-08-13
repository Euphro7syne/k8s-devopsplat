#!/usr/bin/env bash
set -euo pipefail

kubectl apply -f deploy/k3s/namespace.yaml
kubectl apply -f deploy/k3s/server-secret.yaml
kubectl apply -f deploy/k3s/postgres.yaml
kubectl -n ops-platform rollout status statefulset/ops-postgres --timeout=180s

kubectl apply -f deploy/k3s/rbac.yaml
kubectl apply -f deploy/k3s/server-config.yaml
kubectl apply -f deploy/k3s/server.yaml
kubectl apply -f deploy/k3s/web.yaml
kubectl apply -f deploy/k3s/ingress.yaml
kubectl -n ops-platform rollout status deploy/ops-server --timeout=120s
kubectl -n ops-platform rollout status deploy/ops-web --timeout=120s
kubectl -n ops-platform get pods,pvc,service -o wide
