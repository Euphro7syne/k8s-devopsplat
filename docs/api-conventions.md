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

当前 auth 错误码包含：`20001 unauthenticated`、`20002 permission denied`、`20003 rate limited`。登录或 MFA Verify 失败达到配置阈值时返回 HTTP `429` 和错误码 `20003`。

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
GET  /api/v1/namespaces/{namespace}
GET  /api/v1/nodes
GET  /api/v1/nodes/{name}
GET  /api/v1/resources/yaml?kind=&namespace=&name=
PUT  /api/v1/resources/yaml
GET  /api/v1/namespaces/{namespace}/pods
GET  /api/v1/namespaces/{namespace}/pods/{pod}
GET  /api/v1/namespaces/{namespace}/pods/{pod}/logs
GET  /ws/v1/namespaces/{namespace}/pods/{pod}/logs/follow
GET  /api/v1/namespaces/{namespace}/pods/{pod}/yaml
DELETE /api/v1/namespaces/{namespace}/pods/{pod}?confirm=true
POST /api/v1/namespaces/{namespace}/pods/{pod}/restart?confirm=true
GET  /api/v1/namespaces/{namespace}/deployments
GET  /api/v1/namespaces/{namespace}/deployments/{name}
GET  /api/v1/namespaces/{namespace}/deployments/{name}/yaml
POST /api/v1/namespaces/{namespace}/deployments/{name}/scale
POST /api/v1/namespaces/{namespace}/deployments/{name}/restart
GET  /api/v1/namespaces/{namespace}/statefulsets
GET  /api/v1/namespaces/{namespace}/statefulsets/{name}
GET  /api/v1/namespaces/{namespace}/statefulsets/{name}/yaml
POST /api/v1/namespaces/{namespace}/statefulsets/{name}/scale
POST /api/v1/namespaces/{namespace}/statefulsets/{name}/restart
GET  /api/v1/namespaces/{namespace}/daemonsets
GET  /api/v1/namespaces/{namespace}/daemonsets/{name}
GET  /api/v1/namespaces/{namespace}/replicasets
GET  /api/v1/namespaces/{namespace}/replicasets/{name}
GET  /api/v1/namespaces/{namespace}/jobs
GET  /api/v1/namespaces/{namespace}/jobs/{name}
GET  /api/v1/namespaces/{namespace}/cronjobs
GET  /api/v1/namespaces/{namespace}/cronjobs/{name}
POST /api/v1/namespaces/{namespace}/cronjobs/{name}/suspend?confirm=true
POST /api/v1/namespaces/{namespace}/cronjobs/{name}/resume?confirm=true
GET  /api/v1/namespaces/{namespace}/services
GET  /api/v1/namespaces/{namespace}/services/{name}
GET  /api/v1/namespaces/{namespace}/ingresses
GET  /api/v1/namespaces/{namespace}/ingresses/{name}
GET  /api/v1/namespaces/{namespace}/configmaps
GET  /api/v1/namespaces/{namespace}/persistentvolumeclaims
GET  /api/v1/namespaces/{namespace}/persistentvolumeclaims/{name}
GET  /api/v1/persistentvolumes
GET  /api/v1/persistentvolumes/{name}
GET  /api/v1/storageclasses
GET  /api/v1/namespaces/{namespace}/events
GET  /api/v1/logs
GET  /api/v1/audit/logs
```

Pod 静态日志支持 `previous/from/limit/keyword/level`；实时日志使用 WebSocket 转发 Kubernetes `follow=true`。WebSocket 认证通过 `Sec-WebSocket-Protocol: ops-platform.logs.v1, bearer.<access-token>`，不把 access token 放入 URL。

Pod restart 仅接受 ReplicaSet、StatefulSet 或 DaemonSet 管理的 Pod。Job Pod、独立 Pod 和无法识别控制器的 Pod 返回 HTTP `409 / code=10005`；Job 重新运行和删除当前没有 API。

CronJob suspend/resume 仅允许 `operator/admin`，后端强制要求 `confirm=true`，并由审计中间件记录。暂停只影响未来调度，不终止已经开始的 Job；立即运行和删除当前没有 API。

Service 详情优先使用 EndpointSlice，只有在不存在 EndpointSlice 时才回退同名传统 Endpoints，避免端点重复。关联 Pod 同时覆盖 selector 命中和 endpoint targetRef；接口为只读诊断能力，不新增 Service 删除权限。

Ingress 详情返回默认后端与 Host/Path 规则，并校验 Service backend 的 Service 和端口是否存在；重复 Service 只展开一次完整 ServiceDetail。Ingress、Service、EndpointSlice/Endpoints 和 Pod Event 按 UID 过滤关联。公网无 Host/IP 入口不由该读取接口隐式修改，继续按独立部署任务处理。

PVC→PV 通过 PVC volumeName 与 PV claimRef 校验，PV→PVC 依据 claimRef namespace/name/UID 解析且拒绝冲突的 PVC volumeName；详情继续关联实际引用 Claim 的 Pod、顶层 Workload、容器挂载/块设备路径和 Event。卷源只返回安全摘要，CSI `volumeAttributes`、volume handle 和所有 SecretRef 均不进入响应；详情接口只读，不开放删除、扩容或 StorageClass 修改。

Namespace 详情返回 Labels/Finalizers/Conditions、资源数量、非终态 Pod 的有效 requests/limits、Pod/顶层 Workload/Service/Ingress/PVC 和 Namespace 自身 Event；Namespace YAML 支持只读导出，删除无 API。Node 详情返回 roles、地址、污点、Conditions、系统信息、Capacity/Allocatable、有效 Pod requests/limits、声明占比、Pod/顶层 Workload 和 Node 自身 Event。声明占比不等同于实时使用率，Node 写操作与指定调度均无 API。

YAML 更新仅开放 `deployment/statefulset/daemonset/job/cronjob/service/ingress`，由 `operator/admin` 调用并经过审计中间件。ConfigMap 仍必须走配置中心流程，Secret 仍未纳入阶段 1 的通用列表与 YAML 导出。后续启用 Secret 时必须保持列表脱敏、明文读取受角色控制，并同步 Kubernetes RBAC。

`auth.mfa_enabled=true` 时，MFA 登录响应返回 `mfa_required=true`、`mfa_setup_required` 和短时 `mfa_token`，此时不会返回 access/refresh token；首次绑定先调用 `auth/mfa/setup` 获取 TOTP URI，再用 `auth/mfa/verify` 完成绑定并取得 JWT。`auth.mfa_enabled=false` 时，密码校验成功后直接签发 JWT，不进入动态码流程。登录与 MFA Verify 默认启用失败速率限制，达到阈值返回 HTTP `429`。管理员重置 MFA 属于删除类操作，必须携带 `confirm=true`。
