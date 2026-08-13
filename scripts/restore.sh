#!/usr/bin/env bash
set -euo pipefail

DB_DRIVER="${DB_DRIVER:-${OPS_DATABASE_DRIVER:-sqlite}}"
DB_PATH="${DB_PATH:-data/ops-platform.db}"
DATABASE_DSN="${DATABASE_DSN:-${OPS_DATABASE_DSN:-}}"

if [[ -z "${BACKUP_FILE:-}" ]]; then
  echo "BACKUP_FILE is required" >&2
  exit 1
fi

case "${DB_DRIVER}" in
  sqlite|sqlite3)
    mkdir -p "$(dirname "${DB_PATH}")"
    sqlite3 "${DB_PATH}" ".restore '${BACKUP_FILE}'"
    ;;
  postgres|postgresql)
    if [[ -z "${DATABASE_DSN}" ]]; then
      echo "DATABASE_DSN or OPS_DATABASE_DSN is required for PostgreSQL restore" >&2
      exit 1
    fi
    pg_restore \
      --dbname="${DATABASE_DSN}" \
      --clean \
      --if-exists \
      --no-owner \
      --no-privileges \
      "${BACKUP_FILE}"
    ;;
  *)
    echo "unsupported DB_DRIVER: ${DB_DRIVER}" >&2
    exit 1
    ;;
esac
