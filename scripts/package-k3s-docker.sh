#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STAMP="$(date +%Y%m%d%H%M%S)"
OUT_ROOT="${OUT_ROOT:-${ROOT_DIR}/dist/package}"
PKG_DIR="${OUT_ROOT}/ops-platform-k3s-docker-${STAMP}"

mkdir -p "${PKG_DIR}/images" "${PKG_DIR}/deploy" "${PKG_DIR}/test/integration/fixtures"
mkdir -p "${PKG_DIR}/source/ops-platform"

rsync -a \
  --exclude '.cache' \
  --exclude '.git' \
  --exclude '.github' \
  --exclude '.idea' \
  --exclude '.reasonix' \
  --exclude '.DS_Store' \
  --exclude 'bin' \
  --exclude 'data' \
  --exclude 'backup' \
  --exclude 'dist' \
  --exclude 'web/node_modules' \
  --exclude 'web/dist' \
  --exclude 'configs/ops-server.yaml' \
  --exclude 'PROMPT.md' \
  --exclude '参考.md.rtf' \
  --exclude '需要实现的功能.rtf' \
  "${ROOT_DIR}/" "${PKG_DIR}/source/ops-platform/"

if command -v docker >/dev/null 2>&1; then
  docker build -f "${ROOT_DIR}/deploy/docker/ops-server.Dockerfile" -t ops-platform/ops-server:latest "${ROOT_DIR}"
  docker build -f "${ROOT_DIR}/deploy/docker/ops-web.Dockerfile" -t ops-platform/ops-web:latest "${ROOT_DIR}/web"
  docker save ops-platform/ops-server:latest -o "${PKG_DIR}/images/ops-server.tar"
  docker save ops-platform/ops-web:latest -o "${PKG_DIR}/images/ops-web.tar"
else
  touch "${PKG_DIR}/images/.build-on-server"
fi

cp -R "${ROOT_DIR}/deploy/k3s" "${PKG_DIR}/deploy/"
cp "${ROOT_DIR}/test/integration/fixtures/demo-workload.yaml" "${PKG_DIR}/test/integration/fixtures/"
cp "${ROOT_DIR}/scripts/k3s-docker/"*.sh "${PKG_DIR}/"
cp "${ROOT_DIR}/scripts/k3s-docker/README-DEPLOY.md" "${PKG_DIR}/"
chmod +x "${PKG_DIR}/"*.sh

COPYFILE_DISABLE=1 tar -C "${OUT_ROOT}" -czf "${OUT_ROOT}/ops-platform-k3s-docker-${STAMP}.tar.gz" "ops-platform-k3s-docker-${STAMP}"
echo "${OUT_ROOT}/ops-platform-k3s-docker-${STAMP}.tar.gz"
