#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERIFY_MFA_STORAGE="${VERIFY_MFA_STORAGE:-false}"
VERIFY_POSTGRES_RESTART="${VERIFY_POSTGRES_RESTART:-false}"

WORK_ROOT="${SCRIPT_DIR}"
if [[ ! -f "${WORK_ROOT}/test/integration/fixtures/demo-workload.yaml" ]]; then
  WORK_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
cd "${WORK_ROOT}"

echo "[1/2] verifying deployment, database, informer cache and Ingress"
"${SCRIPT_DIR}/verify-deployment.sh"

echo "[2/2] applying P0 demo resources"
"${SCRIPT_DIR}/apply-demo.sh"

if [[ "${VERIFY_MFA_STORAGE}" == "true" ]]; then
  echo "[optional] verifying encrypted MFA storage"
  "${SCRIPT_DIR}/verify-mfa-storage.sh"
else
  echo "[optional] MFA storage skipped (set VERIFY_MFA_STORAGE=true after enrolling a user)"
fi

if [[ "${VERIFY_POSTGRES_RESTART}" == "true" ]]; then
  if [[ "${CONFIRM_RESTART:-}" != "true" ]]; then
    echo "Set CONFIRM_RESTART=true together with VERIFY_POSTGRES_RESTART=true" >&2
    exit 1
  fi
  echo "[optional] verifying PostgreSQL Pod recreation and persistence"
  "${SCRIPT_DIR}/verify-postgres-restart.sh"
  "${SCRIPT_DIR}/verify-deployment.sh"
else
  echo "[optional] PostgreSQL recreation skipped (set VERIFY_POSTGRES_RESTART=true CONFIRM_RESTART=true)"
fi

echo "P0 server-side verification completed"
