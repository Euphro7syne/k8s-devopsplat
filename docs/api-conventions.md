# API 约定

统一前缀：`/api/v1`。

统一响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

错误码分段：

| 范围 | 模块 |
|---|---|
| 10000-19999 | common |
| 20000-29999 | auth |
| 30000-39999 | kubernetes |
| 40000-49999 | configcenter |
| 50000-59999 | release |
| 60000-69999 | logquery |
| 70000-79999 | audit |
| 80000-89999 | ai |
| 90000-99999 | asset |

分页参数统一为 `page` / `page_size`。

当前 common 错误码包含：`10001 internal`、`10002 invalid argument`、`10003 not found`、`10004 service unavailable`、`10005 conflict`。

阶段 1 已实现：

```text
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/mfa/setup
POST /api/v1/auth/mfa/verify
GET  /api/v1/auth/profile
GET  /api/v1/auth/mfa/status
POST /api/v1/auth/mfa/enrollment
POST /api/v1/auth/mfa/enable
POST /api/v1/auth/mfa/disable
GET  /api/v1/roles
GET/POST /api/v1/users
PUT  /api/v1/users/{id}/status
PUT  /api/v1/users/{id}/roles
DELETE /api/v1/users/{id}/mfa?confirm=true
GET  /api/v1/clusters
GET  /api/v1/overview
GET  /api/v1/namespaces
GET  /api/v1/nodes
GET  /api/v1/resources/yaml?kind=&namespace=&name=
PUT  /api/v1/resources/yaml
GET  /api/v1/namespaces/{namespace}/pods
GET  /api/v1/namespaces/{namespace}/pods/{pod}
GET  /api/v1/namespaces/{namespace}/pods/{pod}/logs
GET  /api/v1/namespaces/{namespace}/pods/{pod}/yaml
DELETE /api/v1/namespaces/{namespace}/pods/{pod}?confirm=true
GET  /api/v1/namespaces/{namespace}/deployments
GET  /api/v1/namespaces/{namespace}/deployments/{name}
GET  /api/v1/namespaces/{namespace}/deployments/{name}/yaml
POST /api/v1/namespaces/{namespace}/deployments/{name}/scale
POST /api/v1/namespaces/{namespace}/deployments/{name}/restart
GET  /api/v1/namespaces/{namespace}/statefulsets
GET  /api/v1/namespaces/{namespace}/daemonsets
GET  /api/v1/namespaces/{namespace}/replicasets
GET  /api/v1/namespaces/{namespace}/jobs
GET  /api/v1/namespaces/{namespace}/cronjobs
GET  /api/v1/namespaces/{namespace}/services
GET  /api/v1/namespaces/{namespace}/ingresses
GET  /api/v1/namespaces/{namespace}/configmaps
GET  /api/v1/namespaces/{namespace}/persistentvolumeclaims
GET  /api/v1/persistentvolumes
GET  /api/v1/storageclasses
GET  /api/v1/namespaces/{namespace}/events
GET  /api/v1/logs
GET  /api/v1/audit/logs
```

YAML 更新仅开放 `deployment/statefulset/daemonset/job/cronjob/service/ingress`，由 `operator/admin` 调用并经过审计中间件。ConfigMap 仍必须走配置中心流程，Secret 仍未纳入阶段 1 的通用列表与 YAML 导出。后续启用 Secret 时必须保持列表脱敏、明文读取受角色控制，并同步 Kubernetes RBAC。

MFA 登录响应在需要二次验证时返回 `mfa_required=true`、`mfa_setup_required` 和短时 `mfa_token`，此时不会返回 access/refresh token。首次绑定先调用 `auth/mfa/setup` 获取 TOTP URI，再用 `auth/mfa/verify` 完成绑定并取得 JWT。管理员重置 MFA 属于删除类操作，必须携带 `confirm=true`。
