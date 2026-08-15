# 架构说明

阶段 0 只落工程骨架：Go 模块化单体、Vue 控制台、存储接口、统一响应与错误码。存储层目前实现 SQLite 与 PostgreSQL 两种驱动，本地默认 SQLite，k3s 正式部署默认 PostgreSQL。

后续模块继续按 `handler -> service -> store` 分层推进，跨模块依赖收敛到 `internal/server` 路由与 `internal/store` 接口。

阶段 1 MVP 资源运维链路：

- `internal/auth`：本地 email 登录、JWT access/refresh、RFC 6238 TOTP 绑定与验证、角色校验，以及 Admin-only 用户/角色/MFA 重置管理。
- `internal/resources`：Namespace、Node、Pod、Deployment、StatefulSet、ReplicaSet、DaemonSet、Job、CronJob、Service、Ingress、ConfigMap、Secret、PV/PVC、StorageClass、Event、YAML 与集群概览查询；非敏感只读对象由 `internal/k8s` 的 typed Informer 缓存提供，`resource_mapper` 校验 UID 后解析 Pod 顶层控制器，缓存不可用时回退 Kubernetes API；Secret 只开放 ConfigAdmin/Admin 脱敏元数据和 Admin 单 key 审计明文读取，不进入 Informer 或通用 YAML；Namespace/Node 详情汇总健康状态、声明资源、Pod/顶层 Workload 和相关网络/存储资源，且不混同 P1 实时指标；Service 详情优先通过 EndpointSlice 建立 Endpoint/Pod/Event 关联，无 EndpointSlice 时回退 Endpoints；Ingress 详情继续复用 Service 关联，形成 Ingress→Service→Endpoint→Pod/Event 链路；PVC/PV 详情校验双向绑定并形成 PVC/PV→Pod→顶层 Workload/容器挂载/Event 链路，卷源仅返回安全摘要；YAML 更新仅开放 Workload、Service、Ingress 的受限集合。
- `internal/logquery`：MVP 使用 Kubernetes API 直查 Pod 日志。
- `internal/workload`：Pod 删除与受控 Pod 重启，Deployment/StatefulSet 扩缩容与滚动重启。
- `internal/audit`：写操作审计落库和审计日志查询。

当前阶段的资源写操作开放 Pod 删除与受控 Pod 重启、Deployment/StatefulSet 扩缩容与滚动重启、CronJob 暂停/恢复，以及 `deployment/statefulset/daemonset/job/cronjob/service/ingress` YAML 更新；Secret 单 key 明文读取使用 POST 只是为了审计，不修改 Kubernetes 对象。Secret/ConfigMap 直改、Job/CronJob 删除、CronJob 立即运行、Pod exec、Namespace 删除、Node cordon/drain/污点修改和指定调度等高危或多节点能力仍保持关闭。
