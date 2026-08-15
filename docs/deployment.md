# 部署

默认部署到 `ops-platform` namespace。镜像标签由 Makefile 的 `SERVER_IMAGE`、`WEB_IMAGE` 控制。

## 数据库部署

k3s 默认使用集群内 PostgreSQL：

- `deploy/k3s/postgres.yaml` 创建单副本 `StatefulSet/ops-postgres`。
- PostgreSQL 数据保存在 5Gi `ReadWriteOnce` PVC 中，Pod 重建不会丢失数据。
- `Service/ops-postgres` 只提供集群内访问，不通过 Ingress 或 NodePort 暴露。
- `ops-server` 使用 `OPS_DATABASE_DRIVER=postgres` 和 Secret 中的 `OPS_DATABASE_DSN` 连接数据库。
- PostgreSQL 密码和完整 DSN 位于 `ops-server-secrets`，不能写入 ConfigMap 或提交真实值。

SQLite 仍作为本地开发和轻量回退选项。可选 PVC 清单位于 `deploy/k3s/optional/sqlite-pvc.yaml`，不会被默认的 `kubectl apply -f deploy/k3s` 自动安装。切回 SQLite 时还需修改 `server-config.yaml` 和 `server.yaml` 的数据库配置与数据卷挂载。

PostgreSQL 是单节点 k3s 内的单副本数据库，不提供跨节点高可用。应对 PostgreSQL PVC 定时执行逻辑备份，并将备份复制到节点之外。

## Secret 初始化

认证和数据库敏感配置由 `deploy/k3s/server-secret.yaml` 注入：

- JWT 签名密钥。
- TOTP AES-GCM 主密钥。
- 管理员密码，默认值为 `admin123`。
- PostgreSQL 密码。
- PostgreSQL DSN。

部署前必须替换全部 `change-me-*` 占位符。k3s Docker 验证包可运行：

```bash
scripts/k3s-docker/prepare-config.sh
```

脚本会生成随机认证密钥和 PostgreSQL 密码，并同步生成 DSN；管理员默认密码保持为 `admin123`。生产环境应通过 `OPS_AUTH_LOCAL_ADMIN_PASSWORD` 覆盖该默认值。服务启动时会把配置中的本地管理员密码同步为数据库中的 bcrypt 哈希，因此修改配置或 Secret 后重启 `ops-server` 即可更新已有管理员密码。不要把替换后的 Secret 提交到仓库。

## 部署与检查

`kubernetes.cache` 默认启用进程内 Informer 缓存：`resync_period` 控制周期性重新同步，`sync_timeout` 控制启动阶段等待首次同步的最长时间。首次同步失败不会阻止 API 服务启动，资源读取会回退到 Kubernetes API；修复 RBAC/连接后需重启 `ops-server` 重新建立缓存。Secret 不进入该缓存。`/api/v1/healthz` 的 `resource_cache` 会报告 `ready/not_ready/disabled/unavailable`，其中 `not_ready` 是可用但走 API 回退的状态。

默认 `deploy/k3s/ingress.yaml` 使用无 Host 规则，因此域名或公网 IP 的 80 端口请求都能按 `/api`、`/ws`、`/` 路径进入平台。生产环境如只允许固定域名访问，应在该规则上补回 `host`，并同步配置 DNS、TLS 与入口防火墙；公网可达性仍取决于云安全组、主机防火墙和 Traefik 的 80/443 监听。

```bash
make deploy
kubectl -n ops-platform rollout status statefulset/ops-postgres
kubectl -n ops-platform rollout status deployment/ops-server
kubectl -n ops-platform get pod,pvc,service
```

k3s Docker 验证包提供统一入口 `scripts/k3s-docker/verify-p0.sh`。它默认执行非破坏性的数据库/服务/Informer/Ingress 规则和 demo 检查；需要真实公网验收时传入 `PUBLIC_BASE_URL=http://<公网-ip>`。MFA 存储检查需要已有绑定用户，PostgreSQL Pod 重建必须额外显式确认，详见 `scripts/k3s-docker/README-DEPLOY.md`。

升级已有部署时必须保留集群内现有 `ops-server-secrets`，使用 `PRESERVE_EXISTING_SECRET=true ./deploy.sh`；不要重新生成 PostgreSQL 密码或 TOTP 主密钥。

不要通过 `kubectl describe`、日志或调试命令打印 Secret 值。`ops-server` 启动时会按配置自动执行对应数据库方言的迁移。

阶段 1 调试建议先使用 demo workload，不直接接生产业务 Pod：

```bash
kubectl apply -f test/integration/fixtures/demo-workload.yaml
```

验证资源列表、Pod 日志、删除 Pod、Deployment 扩缩容和重启都通过后，再用只读权限接入真实业务 namespace。
