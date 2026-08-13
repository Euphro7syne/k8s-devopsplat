#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-ops-platform}"

enabled_count="$(
  kubectl -n "${NAMESPACE}" exec statefulset/ops-postgres -- sh -ec \
    'PGPASSWORD="$POSTGRES_PASSWORD" psql -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -Atc "SELECT COUNT(*) FROM users WHERE mfa_secret <> '\'''\'';"'
)"
plaintext_count="$(
  kubectl -n "${NAMESPACE}" exec statefulset/ops-postgres -- sh -ec \
    'PGPASSWORD="$POSTGRES_PASSWORD" psql -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -Atc "SELECT COUNT(*) FROM users WHERE mfa_secret <> '\'''\'' AND mfa_secret NOT LIKE '\''enc:v1:%'\'';"'
)"

if [[ "${enabled_count}" -lt 1 ]]; then
  echo "MFA storage validation failed: no enrolled user found" >&2
  exit 1
fi
if [[ "${plaintext_count}" != "0" ]]; then
  echo "MFA storage validation failed: plaintext-like secrets found: ${plaintext_count}" >&2
  exit 1
fi

echo "MFA-enrolled users: ${enabled_count}"
echo "Unencrypted MFA secrets: ${plaintext_count}"
echo "MFA storage validation: passed"
