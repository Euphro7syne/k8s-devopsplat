#!/usr/bin/env bash
set -euo pipefail

kubectl apply -f deploy/k3s
kubectl -n ops-platform rollout status deploy/ops-server --timeout=120s
kubectl -n ops-platform rollout status deploy/ops-web --timeout=120s
kubectl -n ops-platform get pods -o wide
