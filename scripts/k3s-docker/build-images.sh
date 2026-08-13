#!/usr/bin/env bash
set -euo pipefail

SRC_DIR="${SRC_DIR:-source/ops-platform}"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-postgres:16-alpine}"
DEMO_NGINX_IMAGE="${DEMO_NGINX_IMAGE:-nginx:1.27-alpine}"
DEMO_BUSYBOX_IMAGE="${DEMO_BUSYBOX_IMAGE:-busybox:1.36}"
GO_PROXY="${GOPROXY:-https://goproxy.cn,direct}"

if [[ ! -d "${SRC_DIR}" ]]; then
  echo "source dir not found: ${SRC_DIR}" >&2
  exit 1
fi

docker build \
  --build-arg "GOPROXY=${GO_PROXY}" \
  -f "${SRC_DIR}/deploy/docker/ops-server.Dockerfile" \
  -t ops-platform/ops-server:latest \
  "${SRC_DIR}"
docker build -f "${SRC_DIR}/deploy/docker/ops-web.Dockerfile" -t ops-platform/ops-web:latest "${SRC_DIR}/web"
docker pull "${POSTGRES_IMAGE}"
docker pull "${DEMO_NGINX_IMAGE}"
docker pull "${DEMO_BUSYBOX_IMAGE}"
docker images | grep 'ops-platform/ops-'
docker image inspect "${POSTGRES_IMAGE}" >/dev/null
docker image inspect "${DEMO_NGINX_IMAGE}" >/dev/null
docker image inspect "${DEMO_BUSYBOX_IMAGE}" >/dev/null
