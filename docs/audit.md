# 审计

阶段 0 已接入写请求审计占位中间件，并初始化 `audit_logs` 表。

阶段 1 开始，所有写操作需要落 `user / action / resource / namespace / request_body / ip / time`，审计数据只追加不可改。

已接入 REST 写请求审计：

- `DELETE /api/v1/namespaces/{namespace}/pods/{pod}`
- `POST /api/v1/namespaces/{namespace}/deployments/{name}/scale`
- `POST /api/v1/namespaces/{namespace}/deployments/{name}/restart`

审计查询：`GET /api/v1/audit/logs`，仅 `auditor` / `admin` 可访问。请求体会经过 sanitizer，`password/token/secret/authorization/api_key/private_key` 等字段会打码。
