# RBAC

平台内部角色固定为 `viewer`、`operator`、`configadmin`、`auditor`、`admin`。

阶段 1 已实现本地 email 登录、JWT 会话、TOTP MFA 和接口级角色校验。

最小权限约束：

| 接口能力 | 平台角色 |
|---|---|
| 用户 / 角色管理、重置他人 MFA | admin |
| 资源查看 / Pod 日志 | viewer/operator/configadmin/auditor/admin |
| Secret 元数据、key 和大小查看 | configadmin/admin |
| Secret 单 key 明文读取（`confirm=true`，审计） | admin |
| 删除/重启受控 Pod、扩缩容和重启 Deployment/StatefulSet、重启 DaemonSet、暂停/恢复 CronJob、受限资源 YAML 更新 | operator/admin |
| 查看审计日志 | auditor/admin |

用户管理接口包括 `GET /api/v1/users`、`POST /api/v1/users`、`PUT /api/v1/users/{id}/status`、`PUT /api/v1/users/{id}/roles`、`DELETE /api/v1/users/{id}/mfa?confirm=true` 和 `GET /api/v1/roles`。创建用户只支持本地账号，密码使用 bcrypt hash 存储；管理员不能禁用自己、修改自己的角色或通过用户管理接口重置自己的 MFA。认证中间件会按 access token 中的 user id 回查当前用户状态、角色和 `auth.mfa_enabled` 登录开关，禁用、角色变更和 MFA 策略变化无需等待 JWT 过期。

Kubernetes RBAC 只开放资源只读、`pods/log` 读取、`pods` 删除，以及 `deployments/statefulsets/daemonsets/jobs/cronjobs/services/ingresses` 更新。资源只读覆盖 core/apps/batch/discovery/networking/storage 中 P0 资源：Namespace、Node、Pod、Endpoints、EndpointSlice、Event、ConfigMap、Secret、Service、PVC/PV、Deployment、StatefulSet、ReplicaSet、DaemonSet、Job、CronJob、Ingress、StorageClass。Secret 仅开放 `get/list/watch`，不开放任何写动词；EndpointSlice 同样仅开放 `get/list/watch`，供 Service 和 Ingress 网络关联详情复用。

CronJob 暂停/恢复复用现有 `cronjobs/update`，平台没有开放 `jobs/create`，因此暂不提供“立即运行”；CronJob 删除权限同样没有开放。暂停/恢复均要求显式确认并经过审计。

Pod 没有原地重启 API。平台的 Pod 重启会先确认 Pod 由 ReplicaSet、StatefulSet 或 DaemonSet 管理，再删除 Pod 让控制器重建。Job Pod 不适用该语义：删除运行中 Pod 可能影响失败计数，删除已完成 Pod 也不会重跑 Job，因此返回冲突错误；独立 Pod 同样返回冲突错误，避免把“删除”误当成可恢复的重启。

ConfigMap YAML 仍只读，后续通过配置中心版本发布落地，避免绕开 App/Env/ConfigVersion 审计链路。Secret 列表/详情只允许 `configadmin/admin` 且不包含值；单 key 明文读取只允许 `admin`，强制 `confirm=true` 并使用 POST 进入审计。`pods/exec`、Secret 写、Namespace 删除和 Node 写操作仍未开放。P1 配置中心存储 Secret 配置项时仍必须使用 AES-GCM，并由版本发布链路写入 Kubernetes。
