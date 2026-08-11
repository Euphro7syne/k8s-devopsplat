#!/usr/bin/env bash
set -euo pipefail

SRC_DIR="${SRC_DIR:-source/ops-platform}"

if [[ ! -d "${SRC_DIR}" ]]; then
  echo "source dir not found: ${SRC_DIR}" >&2
  exit 1
fi

docker build -f "${SRC_DIR}/deploy/docker/ops-server.Dockerfile" -t ops-platform/ops-server:latest "${SRC_DIR}"
docker build -f "${SRC_DIR}/deploy/docker/ops-web.Dockerfile" -t ops-platform/ops-web:latest "${SRC_DIR}/web"
docker images | grep 'ops-platform/ops-'
