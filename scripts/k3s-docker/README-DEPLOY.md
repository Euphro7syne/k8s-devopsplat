# ops-platform k3s Docker Runtime 验证包

适用于以 Docker 作为容器运行时的单节点 k3s。节点需要 `bash`、`awk`、`curl`、`docker` 和 `kubectl`，并能拉取基础镜像及 Go/npm 依赖。

## 1. 初始化配置

```bash
tar -xzf ops-platform-k3s-docker-*.tar.gz
cd ops-platform-k3s-docker-*

./prepare-config.sh
```

脚本会生成 JWT、TOTP AES-GCM 和 PostgreSQL 密码，并写入 `deploy/k3s/server-secret.yaml`。默认管理员账号为 `admin@example.com`，默认密码为 `admin123`。

不要重复运行 `prepare-config.sh`，也不要把生成后的 Secret 文件提交到仓库或发送到不受信任的位置。

验证包的 k3s 配置默认：

- PostgreSQL 单副本 StatefulSet + 5Gi PVC。
- `ops-server` 和 `ops-web` 各 1 副本。
- `auth.mfa_enabled: true`，首次登录必须绑定 MFA。

## 2. 构建镜像

```bash
./build-images.sh
```

后端镜像构建默认使用 `https://goproxy.cn,direct` 下载 Go 模块。如果部署节点需要其他代理，可执行：

```bash
GOPROXY=https://your-go-proxy.example,direct ./build-images.sh
```

脚本会构建：

- `ops-platform/ops-server:latest`
- `ops-platform/ops-web:latest`

并预拉取 PostgreSQL 与 demo 验收使用的公共镜像：

- `postgres:16-alpine`
- `nginx:1.27-alpine`
- `busybox:1.36`

确认镜像存在：

```bash
docker images | grep -E 'ops-platform/ops-|postgres|nginx|busybox'
```

如果验证包已包含 `images/ops-server.tar` 和 `images/ops-web.tar`，可改用：

```bash
./load-images.sh
```

## 3. 部署与基础验证

```bash
./deploy.sh
./verify-deployment.sh
```

`deploy.sh` 会先部署并等待 PostgreSQL，再部署后端和前端。`verify-deployment.sh` 会检查：

- PostgreSQL StatefulSet、ops-server、ops-web 均完成 rollout。
- PostgreSQL 六张基础表已创建。
- 五个内置角色已初始化。
- `/api/v1/healthz` 返回数据库健康状态。
- Pod、PVC 和 Service 状态。

## 4. MFA 场景 A

在节点执行端口转发：

```bash
kubectl -n ops-platform port-forward svc/ops-web 18080:80
```

浏览器访问 `http://<节点可访问地址>:18080`。如果浏览器就在节点本机，使用 `http://127.0.0.1:18080`。

使用默认管理员账号密码登录：

```text
账号：admin@example.com
密码：admin123
```

按顺序验证：

1. 密码校验成功后进入首次 MFA 绑定页，不应直接进入控制台。
2. 使用兼容 TOTP 的身份验证器扫描二维码。
3. 输入 6 位动态码，完成绑定并进入控制台。
4. 退出登录并再次输入账号密码。
5. 第二次登录只要求动态码，不应再次展示绑定二维码。
6. 进入“安全设置”，确认平台显示强制 MFA，且关闭功能不可用。

完成绑定后，在节点执行：

```bash
./verify-mfa-storage.sh
```

期望输出：至少一个已绑定用户，未加密 MFA Secret 数量为 0。脚本不会输出 TOTP 种子或密文内容。

## 5. PostgreSQL Pod 重建持久性

该步骤会删除并由 StatefulSet 自动重建 PostgreSQL Pod，但不会删除 PVC。确认当前没有发布或写操作后执行：

```bash
CONFIRM_RESTART=true ./verify-postgres-restart.sh
./verify-deployment.sh
```

期望用户数量在 Pod 重建前后保持一致，数据库和后端重新恢复健康。

## 6. Demo 资源功能验证

```bash
./apply-demo.sh
```

随后在控制台使用 `demo-app` namespace 验证：

1. Deployment 详情、YAML 编辑、扩缩容和滚动重启。
2. StatefulSet 详情、Headless Service、YAML 编辑、扩缩容和滚动重启。
3. DaemonSet 详情、调度信息、YAML 编辑和滚动重启；DaemonSet 不提供扩缩容。
4. 保存 Deployment/StatefulSet/DaemonSet YAML 后会出现配置生效提示；选择立即重启时，YAML 更新与 restart 分别生成审计记录。
5. 对 Deployment、StatefulSet 或 DaemonSet 管理的 Pod 执行重启，确认原 Pod 被删除并由控制器创建替代 Pod。
6. Pod 日志、Pod 显式删除与操作审计。

真实业务 namespace 首次只做查看和日志验证。

## 7. 常用排查命令

```bash
kubectl -n ops-platform get pod,pvc,service -o wide
kubectl -n ops-platform logs deploy/ops-server --tail=100
kubectl -n ops-platform logs statefulset/ops-postgres --tail=100
kubectl -n ops-platform describe pod ops-server-<pod-suffix>
```

不要输出、解码或复制 `ops-server-secrets` 的内容到日志、聊天或工单中。
