#!/usr/bin/env bash
set -euo pipefail

kubectl apply -f test/integration/fixtures/demo-workload.yaml
kubectl -n demo-app rollout status deploy/nginx-demo --timeout=120s
kubectl -n demo-app rollout status deploy/log-demo --timeout=120s
kubectl -n demo-app rollout status statefulset/stateful-demo --timeout=120s
kubectl -n demo-app rollout status daemonset/daemon-demo --timeout=120s
kubectl -n demo-app get pods -o wide
