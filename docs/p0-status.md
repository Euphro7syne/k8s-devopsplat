# P0 完成度与开发顺序

本文件按 README 的 P0 清单逐项记录实际完成度。状态定义：

- **完成**：API、后端、前端、RBAC/审计和必要测试代码已形成闭环。
- **部分完成**：已有列表或部分接口，但详情、关联、操作、安全边界或测试尚未完整。
- **待验收**：代码闭环已完成，但最新版本尚未完成服务器部署和真实 k3s 验收。
- **未开始**：只有设计占位或空包。

## 基础能力

| 能力 | 状态 | 已有内容 | 剩余内容 |
|---|---|---|---|
| 本地 email/JWT/MFA | 完成 | 登录、refresh、TOTP、限流、管理员重置、前端、安全存储 | LDAP/OIDC 属可选扩展，不阻塞本地认证 MVP |
| 平台 RBAC | 完成 | Viewer/Operator/ConfigAdmin/Auditor/Admin，前后端路由控制 | 新增能力时继续按矩阵收敛权限 |
| PostgreSQL/SQLite | 待验收 | 双驱动、迁移、k3s PostgreSQL StatefulSet/PVC | 最新版本统一真实验收 |
| 操作审计 | 完成 | 写请求横切审计、查询页、敏感请求体脱敏 | 新写接口必须同步加入审计检查 |
| 集群概览 | 部分完成 | 节点、Namespace、Pod、异常 Pod、allocatable | CPU/Memory 实际使用率属于 P1 metrics-server |
| 多集群 | 预留 | in-cluster 标识和 UI 禁用切换入口 | P2 正式启用；当前不开发 |

## 资源覆盖矩阵

| 资源 | 状态 | 当前能力 | 主要缺口 |
|---|---|---|---|
| Namespace | 待验收 | 列表、详情、Labels/Finalizers/Conditions、自身 Event、资源计数、有效 Pod requests/limits、Pod/Workload/Service/Ingress/PVC 关联和只读 YAML | 最新版本真实 k3s 验收；删除保持关闭 |
| Node | 待验收 | 列表/YAML、roles、Conditions、地址、污点、系统信息、Capacity/Allocatable、有效 Pod requests/limits、Pod/Workload/Event 关联 | 最新版本真实 k3s 验收；实时使用率属于 P1，节点写操作和指定调度暂不开放 |
| Pod | 待验收 | 列表、诊断详情、控制器链、Event、YAML、删除、ReplicaSet/StatefulSet/DaemonSet 受控重启 | 真实日志/WebSocket 与集群控制器行为验收；Job Pod 重启保持 409 |
| Deployment | 待验收 | 详情、Conditions、Deployment→ReplicaSet→Pod/Event 联动、Pod 日志下钻、YAML、扩缩容、滚动重启 | 最新版本真实 k3s 验收 |
| StatefulSet | 待验收 | 详情、PVC 模板、YAML、扩缩容、RollingUpdate 重启、OnDelete 边界 | 最新版本真实 k3s 验收 |
| ReplicaSet | 待验收 | 详情、所属 Deployment、直属 Pod/Event、Pod 日志下钻、YAML | 最新版本真实 k3s 验收；直接扩缩容/删除保持关闭 |
| DaemonSet | 待验收 | 详情、调度信息、YAML、滚动重启、OnDelete 边界 | 最新版本真实 k3s 验收 |
| Job | 待验收 | 列表、YAML、执行策略/Conditions/CronJob owner 详情、直属 Pod/Event 和日志下钻；删除/重新运行关闭，Job Pod 重启返回 409 | 最新版本真实 k3s 验收 |
| CronJob | 待验收 | 调度/时区/并发/历史保留/JobTemplate 详情、历史 Job/Pod/Event 和日志下钻；暂停/恢复受控开放，立即运行/删除关闭 | 最新版本真实 k3s 验收 |
| Service | 待验收 | 列表/YAML、流量策略/结构化端口详情、EndpointSlice 优先与 Endpoints 回退、selector/targetRef Pod、Event 和日志下钻 | 最新版本真实 k3s 验收 |
| Ingress | 待验收 | 列表/YAML、默认与规则后端、TLS、Service/端口校验、Service→EndpointSlice/Endpoints→Pod/Event 和日志下钻；部署清单提供无 Host 公网入口 | 最新版本真实 k3s/Traefik、云安全组和主机防火墙验收 |
| PV/PVC/StorageClass | 待验收 | 列表/YAML、PVC/PV 双向详情、有效绑定校验、Pod/Workload、容器挂载/块设备路径、Event 和安全卷源摘要；删除/扩容/StorageClass 修改关闭 | 最新版本真实 k3s 验收 |
| ConfigMap | 部分完成 | 列表、Key、只读 YAML | 写入必须等 P1 配置中心版本发布，不开放直接编辑 |
| Secret | 待验收 | ConfigAdmin/Admin 脱敏列表与详情、key/大小元数据；Admin 单 key `confirm=true` 审计明文读取；通用 YAML 与写入关闭 | 最新版本真实 k3s 验收；写入走 P1 配置中心 |
| Event | 完成 | 资源页独立 Namespace Event 视图、按 involved kind/name 筛选，以及 Namespace/Node/Pod/Deployment/ReplicaSet/Job/CronJob/Service/Ingress/PVC/PV 详情联动 | 聚合、去重、告警和 webhook 属于 P1 |

## Pod 日志

| 能力 | 状态 | 说明 |
|---|---|---|
| 当前容器静态日志 | 待验收 | Kubernetes API，行数/时间/关键字/级别筛选；已有 k3d 用例，尚未实际运行 |
| 上一次容器日志 | 待验收 | `previous=true`，fixture 通过真实容器重启生成 previous 日志并校验；尚未实际运行 |
| 实时日志 | 待验收 | WebSocket 转发 `follow=true`，已有 k3d 用例，尚未实际运行 |
| 日志脱敏 | 待验收 | REST/WebSocket 输出统一调用 sanitizer，previous 用例同时校验 token 脱敏；尚未实际运行 |
| 长期历史日志 | 未开始（P1） | Fluent Bit → Loki，支持跨 Pod、跨轮转、按时间检索 |

## 不遗漏的后续顺序

1. Namespace/Node、Service/Ingress、PV/PVC 和 Secret 安全只读已完成代码闭环并等待真实 k3s 验收；多节点指定调度只保留扩展位。
2. informer/cache 与资源关联 mapper 已完成代码闭环：非敏感只读资源走 typed Informer，Secret 不缓存，缓存未就绪回退 API。
3. 本地门禁和统一验收代码已完成；在具备 Docker/k3d/kubectl 的环境执行 k3d、服务器镜像构建和真实 k3s 验收。integration 会强制确认 `resource_cache=ready`，并覆盖缓存 watch/RBAC、Secret 防泄漏、单 key 明文权限/审计、previous 日志与其他资源真实行为。
4. Ingress 公网 IP 无 Host 访问规则已完成代码闭环，纳入统一真实环境验收。
5. P0 验收完成后进入 P1：配置中心 → 发布执行器 → Loki → metrics/event 规则 → AI 规则诊断与受控修复。

所有 AI 修复动作都必须经过 RBAC、二次确认和审计；模型不得直接执行任意 Kubernetes 修改。

## 当前验收基线

2026-08-13 已通过 `make test`、`make lint`、`make build`、目标包 race 测试、integration 编译、OpenAPI/k3s YAML 解析、全部 Shell 语法和 `git diff --check`。当前执行机没有 Docker、k3d 或 kubectl，真实集群与公网入口不得标记为已通过。

服务器统一入口为 `scripts/k3s-docker/verify-p0.sh`。默认执行非破坏性的部署健康、Informer ready、无 Host Ingress 和 demo 检查；MFA 存储需要已有绑定用户，PostgreSQL Pod 重建必须显式设置 `VERIFY_POSTGRES_RESTART=true CONFIRM_RESTART=true`。公网入口通过 `PUBLIC_BASE_URL=http://<公网-ip>` 验证。
