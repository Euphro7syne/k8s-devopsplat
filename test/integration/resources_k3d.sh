#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${REPO_ROOT}"

TEMP_ROOT="${TMPDIR:-/tmp}"
TEMP_ROOT="${TEMP_ROOT%/}"
TEST_TMP_DIR="$(mktemp -d "${TEMP_ROOT}/ops-platform-it.XXXXXX")"
CLUSTER_NAME="${OPS_INTEGRATION_CLUSTER_NAME:-ops-platform-it-$$}"
SERVER_PORT=$((18080 + ($$ % 1000)))
SERVER_LISTEN="127.0.0.1:${SERVER_PORT}"
SERVER_URL="http://${SERVER_LISTEN}"
KUBECONFIG_PATH="${TEST_TMP_DIR}/kubeconfig.yaml"
SERVER_CONFIG="${TEST_TMP_DIR}/ops-server.yaml"
SERVER_LOG="${TEST_TMP_DIR}/ops-server.log"
SERVER_PID=""
CLUSTER_CREATED=false

cleanup() {
  local status=$?
  trap - EXIT

  if [[ -n "${SERVER_PID}" ]]; then
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi

  if [[ "${CLUSTER_CREATED}" == "true" && "${OPS_INTEGRATION_KEEP_CLUSTER:-false}" != "true" ]]; then
    k3d cluster delete "${CLUSTER_NAME}" >/dev/null 2>&1 || true
  fi

  if [[ ${status} -ne 0 && -f "${SERVER_LOG}" ]]; then
    echo "ops-server integration log:" >&2
    sed -n '1,240p' "${SERVER_LOG}" >&2
  fi

  case "${TEST_TMP_DIR}" in
    "${TEMP_ROOT}"/ops-platform-it.*)
      rm -rf -- "${TEST_TMP_DIR}"
      ;;
  esac

  exit "${status}"
}
trap cleanup EXIT

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 1
  fi
}

require_command curl
require_command docker
require_command go
require_command k3d
require_command kubectl

if ! docker info >/dev/null 2>&1; then
  echo "Docker is not available; start Docker before running make test-integration" >&2
  exit 1
fi

if k3d cluster list --no-headers 2>/dev/null | awk '{print $1}' | grep -Fxq "${CLUSTER_NAME}"; then
  echo "refusing to reuse existing k3d cluster: ${CLUSTER_NAME}" >&2
  exit 1
fi

echo "creating temporary k3d cluster ${CLUSTER_NAME}"
k3d cluster create "${CLUSTER_NAME}" \
  --servers 1 \
  --agents 0 \
  --wait \
  --timeout 120s \
  --kubeconfig-update-default=false \
  --kubeconfig-switch-context=false \
  --k3s-arg "--disable=traefik@server:0"
CLUSTER_CREATED=true

k3d kubeconfig get "${CLUSTER_NAME}" > "${KUBECONFIG_PATH}"
kubectl --kubeconfig "${KUBECONFIG_PATH}" apply -f "${REPO_ROOT}/test/integration/fixtures/demo-workload.yaml"

sed \
  -e "s|__LISTEN__|${SERVER_LISTEN}|g" \
  -e "s|__KUBECONFIG__|${KUBECONFIG_PATH}|g" \
  -e "s|__DATABASE_PATH__|${TEST_TMP_DIR}/ops-platform.db|g" \
  "${REPO_ROOT}/test/integration/ops-server.yaml.tpl" > "${SERVER_CONFIG}"

export GOCACHE="${GOCACHE:-${REPO_ROOT}/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-${REPO_ROOT}/.cache/go-mod}"
mkdir -p "${GOCACHE}" "${GOMODCACHE}"

go build -o "${TEST_TMP_DIR}/ops-server" ./cmd/ops-server
"${TEST_TMP_DIR}/ops-server" -config "${SERVER_CONFIG}" > "${SERVER_LOG}" 2>&1 &
SERVER_PID=$!

for _ in $(seq 1 60); do
  if curl -fsS "${SERVER_URL}/api/v1/healthz" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "${SERVER_PID}" 2>/dev/null; then
    echo "ops-server exited before becoming healthy" >&2
    exit 1
  fi
  sleep 1
done

if ! curl -fsS "${SERVER_URL}/api/v1/healthz" >/dev/null; then
  echo "ops-server did not become healthy within 60 seconds" >&2
  exit 1
fi

echo "running resource API integration tests"
OPS_INTEGRATION_BASE_URL="${SERVER_URL}" \
OPS_INTEGRATION_ADMIN_EMAIL="integration-admin@example.invalid" \
OPS_INTEGRATION_ADMIN_PASSWORD="integration-test-password" \
KUBECONFIG="${KUBECONFIG_PATH}" \
go test -tags=integration -count=1 -timeout=12m ./test/integration
