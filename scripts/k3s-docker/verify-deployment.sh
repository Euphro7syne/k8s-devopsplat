#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-ops-platform}"
VERIFY_PORT="${VERIFY_PORT:-18081}"

kubectl -n "${NAMESPACE}" rollout status statefulset/ops-postgres --timeout=180s
kubectl -n "${NAMESPACE}" rollout status deployment/ops-server --timeout=120s
kubectl -n "${NAMESPACE}" rollout status deployment/ops-web --timeout=120s

table_count="$(
  kubectl -n "${NAMESPACE}" exec statefulset/ops-postgres -- sh -ec \
    'PGPASSWORD="$POSTGRES_PASSWORD" psql -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -Atc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '\''public'\'' AND table_name IN ('\''schema_migrations'\'', '\''users'\'', '\''roles'\'', '\''user_roles'\'', '\''clusters'\'', '\''audit_logs'\'');"'
)"
if [[ "${table_count}" != "6" ]]; then
  echo "PostgreSQL schema validation failed: expected 6 tables, got ${table_count}" >&2
  exit 1
fi

role_count="$(
  kubectl -n "${NAMESPACE}" exec statefulset/ops-postgres -- sh -ec \
    'PGPASSWORD="$POSTGRES_PASSWORD" psql -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -Atc "SELECT COUNT(*) FROM roles;"'
)"
if [[ "${role_count}" != "5" ]]; then
  echo "PostgreSQL seed validation failed: expected 5 roles, got ${role_count}" >&2
  exit 1
fi

port_forward_log="$(mktemp)"
kubectl -n "${NAMESPACE}" port-forward service/ops-server "${VERIFY_PORT}:8080" >"${port_forward_log}" 2>&1 &
port_forward_pid=$!
cleanup() {
  kill "${port_forward_pid}" >/dev/null 2>&1 || true
  rm -f "${port_forward_log}"
}
trap cleanup EXIT

health_response=""
for _ in $(seq 1 20); do
  if health_response="$(curl --fail --silent --show-error "http://127.0.0.1:${VERIFY_PORT}/api/v1/healthz" 2>/dev/null)"; then
    break
  fi
  sleep 1
done
if [[ -z "${health_response}" ]]; then
  echo "ops-server health validation failed" >&2
  exit 1
fi

echo "PostgreSQL tables: ${table_count}/6"
echo "Seed roles: ${role_count}/5"
echo "ops-server health: ${health_response}"
kubectl -n "${NAMESPACE}" get pods,pvc,service -o wide
