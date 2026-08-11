#!/usr/bin/env bash
set -euo pipefail

SECRET_FILE="${SECRET_FILE:-deploy/k3s/server-secret.yaml}"

if [[ ! -f "${SECRET_FILE}" ]]; then
  echo "secret file not found: ${SECRET_FILE}" >&2
  exit 1
fi

if command -v openssl >/dev/null 2>&1; then
  JWT_SECRET="$(openssl rand -hex 32)"
  MFA_SECRET_KEY="$(openssl rand -hex 32)"
  ADMIN_PASSWORD="$(openssl rand -hex 12)"
else
  JWT_SECRET="$(date +%s%N)-replace-me"
  MFA_SECRET_KEY="$(date +%s%N)-mfa-replace-me"
  ADMIN_PASSWORD="replace-me-before-deploy"
fi

TMP_FILE="$(mktemp)"
awk \
  -v jwt_secret="${JWT_SECRET}" \
  -v mfa_secret_key="${MFA_SECRET_KEY}" \
  -v admin_password="${ADMIN_PASSWORD}" \
  '{
    gsub(/jwt-secret: "change-me-jwt-secret"/, "jwt-secret: \"" jwt_secret "\"")
    gsub(/mfa-secret-key: "change-me-mfa-secret-key"/, "mfa-secret-key: \"" mfa_secret_key "\"")
    gsub(/local-admin-password: "change-me-admin-password"/, "local-admin-password: \"" admin_password "\"")
    print
  }' "${SECRET_FILE}" > "${TMP_FILE}"
cat "${TMP_FILE}" > "${SECRET_FILE}"
rm -f "${TMP_FILE}"

echo "认证 Secret 已写入 ${SECRET_FILE}"
echo "登录账号：admin@example.com"
echo "临时密码：${ADMIN_PASSWORD}"
