#!/usr/bin/env bash
set -euo pipefail

DB_DRIVER="${DB_DRIVER:-${OPS_DATABASE_DRIVER:-sqlite}}"
DB_PATH="${DB_PATH:-data/ops-platform.db}"
DATABASE_DSN="${DATABASE_DSN:-${OPS_DATABASE_DSN:-}}"
BACKUP_DIR="${BACKUP_DIR:-backup}"
STAMP="$(date +%Y%m%d%H%M%S)"

mkdir -p "${BACKUP_DIR}"

case "${DB_DRIVER}" in
  sqlite|sqlite3)
    sqlite3 "${DB_PATH}" ".backup '${BACKUP_DIR}/ops-platform-${STAMP}.db'"
    ;;
  postgres|postgresql)
    if [[ -z "${DATABASE_DSN}" ]]; then
      echo "DATABASE_DSN or OPS_DATABASE_DSN is required for PostgreSQL backup" >&2
      exit 1
    fi
    pg_dump \
      --dbname="${DATABASE_DSN}" \
      --format=custom \
      --no-owner \
      --no-privileges \
      --file="${BACKUP_DIR}/ops-platform-${STAMP}.dump"
    ;;
  *)
    echo "unsupported DB_DRIVER: ${DB_DRIVER}" >&2
    exit 1
    ;;
esac
