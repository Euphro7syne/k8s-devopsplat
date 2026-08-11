#!/usr/bin/env bash
set -euo pipefail

CONFIG_FILE="${CONFIG_FILE:-deploy/k3s/server-config.yaml}"

if [[ ! -f "${CONFIG_FILE}" ]]; then
  echo "config file not found: ${CONFIG_FILE}" >&2
  exit 1
fi

if command -v openssl >/dev/null 2>&1; then
  JWT_SECRET="$(openssl rand -hex 32)"
  ADMIN_PASSWORD="$(openssl rand -hex 12)"
else
  JWT_SECRET="$(date +%s%N)-replace-me"
  ADMIN_PASSWORD="replace-me-before-deploy"
fi

TMP_FILE="$(mktemp)"
awk \
  -v jwt_secret="${JWT_SECRET}" \
  -v admin_password="${ADMIN_PASSWORD}" \
  '{
    gsub(/jwt_secret: "change-me-placeholder"/, "jwt_secret: \"" jwt_secret "\"")
    gsub(/password: "change-me-admin-password"/, "password: \"" admin_password "\"")
    print
  }' "${CONFIG_FILE}" > "${TMP_FILE}"
cat "${TMP_FILE}" > "${CONFIG_FILE}"
rm -f "${TMP_FILE}"

echo "配置已写入 ${CONFIG_FILE}"
echo "登录账号：admin@example.com"
echo "临时密码：${ADMIN_PASSWORD}"
