# 审计

阶段 0 已接入写请求审计占位中间件，并初始化 `audit_logs` 表。

阶段 1 开始，所有写操作需要落 `user / action / resource / namespace / request_body / ip / time`，审计数据只追加不可改。

已接入 REST 写请求审计：

- `POST /api/v1/auth/mfa/verify`
- `POST /api/v1/auth/mfa/enrollment`
- `POST /api/v1/auth/mfa/enable`
- `POST /api/v1/auth/mfa/disable`
- `DELETE /api/v1/users/{id}/mfa`
- `PUT /api/v1/resources/yaml`
- `DELETE /api/v1/namespaces/{namespace}/pods/{pod}`
- `POST /api/v1/namespaces/{namespace}/pods/{pod}/restart`
- `POST /api/v1/namespaces/{namespace}/deployments/{name}/scale`
- `POST /api/v1/namespaces/{namespace}/deployments/{name}/restart`
- `POST /api/v1/namespaces/{namespace}/statefulsets/{name}/scale`
- `POST /api/v1/namespaces/{namespace}/statefulsets/{name}/restart`
- `POST /api/v1/namespaces/{namespace}/daemonsets/{name}/restart`
- `POST /api/v1/namespaces/{namespace}/secrets/{name}/values/{key}?confirm=true`

Secret 明文读取被建模为写式 POST，虽然不修改 Kubernetes 对象，也必须记录操作者、路径、namespace、来源 IP 和时间。值不放在请求体中，审计 `request_body` 为空；审计记录只保留路由模板中的 key 参数位置，不记录响应明文。

审计查询：`GET /api/v1/audit/logs`，仅 `auditor` / `admin` 可访问。请求体会经过 sanitizer，`password/token/secret/authorization/api_key/private_key` 以及 MFA 动态码等字段会打码。
