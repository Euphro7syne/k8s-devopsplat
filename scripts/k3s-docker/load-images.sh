#!/usr/bin/env bash
set -euo pipefail

if [[ ! -f images/ops-server.tar || ! -f images/ops-web.tar ]]; then
  echo "镜像 tar 不存在。这个包是在没有本机 Docker 的环境生成的，请在服务器上先执行：./build-images.sh" >&2
  exit 1
fi

docker load -i images/ops-server.tar
docker load -i images/ops-web.tar
docker pull postgres:16-alpine
docker pull nginx:1.27-alpine
docker pull busybox:1.36
docker images | grep 'ops-platform/ops-'
