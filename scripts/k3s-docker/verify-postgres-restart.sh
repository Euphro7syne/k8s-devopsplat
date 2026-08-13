#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-ops-platform}"

if [[ "${CONFIRM_RESTART:-}" != "true" ]]; then
  echo "Set CONFIRM_RESTART=true to confirm PostgreSQL Pod restart validation" >&2
  exit 1
fi

user_count_before="$(
  kubectl -n "${NAMESPACE}" exec statefulset/ops-postgres -- sh -ec \
    'PGPASSWORD="$POSTGRES_PASSWORD" psql -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -Atc "SELECT COUNT(*) FROM users;"'
)"

kubectl -n "${NAMESPACE}" delete pod ops-postgres-0
kubectl -n "${NAMESPACE}" wait --for=condition=Ready pod/ops-postgres-0 --timeout=180s

user_count_after="$(
  kubectl -n "${NAMESPACE}" exec statefulset/ops-postgres -- sh -ec \
    'PGPASSWORD="$POSTGRES_PASSWORD" psql -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -Atc "SELECT COUNT(*) FROM users;"'
)"

if [[ "${user_count_before}" != "${user_count_after}" ]]; then
  echo "PostgreSQL persistence validation failed: users ${user_count_before} -> ${user_count_after}" >&2
  exit 1
fi

echo "PostgreSQL Pod restart completed"
echo "User count persisted: ${user_count_after}"
kubectl -n "${NAMESPACE}" get pod/ops-postgres-0 pvc -o wide
