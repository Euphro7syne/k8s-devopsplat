#!/usr/bin/env bash
set -euo pipefail

SECRET_FILE="${SECRET_FILE:-deploy/k3s/server-secret.yaml}"

if [[ ! -f "${SECRET_FILE}" ]]; then
  echo "secret file not found: ${SECRET_FILE}" >&2
  exit 1
fi

for placeholder in \
  change-me-jwt-secret \
  change-me-mfa-secret-key \
  change-me-postgres-password; do
  if ! grep -Fq "${placeholder}" "${SECRET_FILE}"; then
    echo "secret file is already initialized or missing placeholder: ${placeholder}" >&2
    exit 1
  fi
done

if command -v openssl >/dev/null 2>&1; then
  JWT_SECRET="$(openssl rand -hex 32)"
  MFA_SECRET_KEY="$(openssl rand -hex 32)"
  POSTGRES_PASSWORD="$(openssl rand -hex 24)"
else
  JWT_SECRET="$(date +%s%N)-replace-me"
  MFA_SECRET_KEY="$(date +%s%N)-mfa-replace-me"
  POSTGRES_PASSWORD="replace-me-postgres-before-deploy"
fi

DATABASE_DSN="postgres://ops_platform:${POSTGRES_PASSWORD}@ops-postgres:5432/ops_platform?sslmode=disable"

TMP_FILE="$(mktemp)"
awk \
  -v jwt_secret="${JWT_SECRET}" \
  -v mfa_secret_key="${MFA_SECRET_KEY}" \
  -v postgres_password="${POSTGRES_PASSWORD}" \
  -v database_dsn="${DATABASE_DSN}" \
  '{
    gsub(/jwt-secret: "change-me-jwt-secret"/, "jwt-secret: \"" jwt_secret "\"")
    gsub(/mfa-secret-key: "change-me-mfa-secret-key"/, "mfa-secret-key: \"" mfa_secret_key "\"")
    gsub(/postgres-password: "change-me-postgres-password"/, "postgres-password: \"" postgres_password "\"")
    gsub(/database-dsn: "postgres:\/\/ops_platform:change-me-postgres-password@ops-postgres:5432\/ops_platform\?sslmode=disable"/, "database-dsn: \"" database_dsn "\"")
    print
  }' "${SECRET_FILE}" > "${TMP_FILE}"
cat "${TMP_FILE}" > "${SECRET_FILE}"
rm -f "${TMP_FILE}"

echo "认证与数据库 Secret 已写入 ${SECRET_FILE}"
echo "登录账号：admin@example.com"
echo "默认密码：admin123"
