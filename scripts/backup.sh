#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${DB_PATH:-data/ops-platform.db}"
BACKUP_DIR="${BACKUP_DIR:-backup}"
STAMP="$(date +%Y%m%d%H%M%S)"

mkdir -p "${BACKUP_DIR}"
sqlite3 "${DB_PATH}" ".backup '${BACKUP_DIR}/ops-platform-${STAMP}.db'"
