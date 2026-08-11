# 架构说明

阶段 0 只落工程骨架：Go 模块化单体、Vue 控制台、SQLite 存储接口、统一响应与错误码。

后续模块继续按 `handler -> service -> store` 分层推进，跨模块依赖收敛到 `internal/server` 路由与 `internal/store` 接口。

阶段 1 MVP 资源运维链路：

- `internal/auth`：本地 email 登录、JWT access/refresh、角色校验。
- `internal/resources`：Namespace、Node、Pod、Deployment、StatefulSet、ReplicaSet、DaemonSet、Job、CronJob、Service、Ingress、ConfigMap、PV/PVC、StorageClass、Event、YAML 与集群概览只读查询。
- `internal/logquery`：MVP 使用 Kubernetes API 直查 Pod 日志。
- `internal/workload`：Pod 删除、Deployment 扩缩容、Deployment 重启。
- `internal/audit`：写操作审计落库和审计日志查询。

当前阶段的资源写操作只开放 Pod 删除、Deployment 扩缩容和 Deployment 重启；Secret、Pod exec、Namespace 删除等高危能力仍保持关闭。
