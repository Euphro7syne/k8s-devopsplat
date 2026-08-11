# RBAC

平台内部角色固定为 `viewer`、`operator`、`configadmin`、`auditor`、`admin`。

阶段 1 已实现本地 email 登录、JWT 会话、TOTP MFA 和接口级角色校验。

最小权限约束：

| 接口能力 | 平台角色 |
|---|---|
| 用户 / 角色管理、重置他人 MFA | admin |
| 资源查看 / Pod 日志 | viewer/operator/configadmin/auditor/admin |
| 删除 Pod、扩缩容、重启 Deployment、受限资源 YAML 更新 | operator/admin |
| 查看审计日志 | auditor/admin |

用户管理接口包括 `GET /api/v1/users`、`POST /api/v1/users`、`PUT /api/v1/users/{id}/status`、`PUT /api/v1/users/{id}/roles`、`DELETE /api/v1/users/{id}/mfa?confirm=true` 和 `GET /api/v1/roles`。创建用户只支持本地账号，密码使用 bcrypt hash 存储；管理员不能禁用自己、修改自己的角色或通过用户管理接口重置自己的 MFA。认证中间件会按 access token 中的 user id 回查当前用户状态、角色和 MFA 绑定状态，禁用、角色变更和 MFA 策略变化无需等待 JWT 过期。

Kubernetes RBAC 只开放资源只读、`pods/log` 读取、`pods` 删除，以及 `deployments/statefulsets/daemonsets/jobs/cronjobs/services/ingresses` 更新。资源只读覆盖 core/apps/batch/networking/storage 中 P0 资源：Namespace、Node、Pod、Event、ConfigMap、Service、PVC/PV、Deployment、StatefulSet、ReplicaSet、DaemonSet、Job、CronJob、Ingress、StorageClass。

ConfigMap YAML 仍只读，后续通过配置中心版本发布落地，避免绕开 App/Env/ConfigVersion 审计链路。`pods/exec`、Secret 读写、Namespace 删除、Node 写操作仍未开放。后续如果加入 Secret 管理，平台内必须先做 AES-GCM 加密存储、列表脱敏和 Admin/ConfigAdmin 权限约束，再调整 Kubernetes RBAC。
