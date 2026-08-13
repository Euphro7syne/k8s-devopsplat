# 项目总体开发 Checklist

> 盘点日期：2026-08-13
> 权威范围：`AGENTS.md`、根 `README.md`、仓库内全部 README、`docs/`、OpenAPI、后端路由、前端页面、测试与 k3s 清单。
> 状态说明：`[x]` 已完成；`[-]` 部分完成或代码完成待真实环境验收；`[ ]` 未开始；`[R]` 仅预留。

## 1. 当前结论

- 当前总体阶段：**阶段 1 / P0 资源运维 MVP，进行中**。
- 阶段 0 工程骨架已经完成，本地构建与测试链路可用。
- P0 已形成认证、RBAC、审计、Pod 诊断/日志和主要 Workload 操作主链路。
- P0 的网络、存储、Namespace/Node 详情均已形成代码闭环并等待真实 k3s 验收；尚未完成的核心内容是 Secret 安全只读、informer/cache、统一 k3d/真实 k3s 验收和公网 Ingress 入口修复。
- P1 的配置中心、发布、Loki、监控/事件规则和 AI 诊断仍未正式开发；相关模块当前主要是空包或设计占位。
- 多集群与指定 Pod 调度节点只保留扩展位，当前 MVP 不实施；未来接入多节点 k8s 时再正式启用。

## 2. 总体阶段 Checklist

### 阶段 0：工程骨架

- [x] Go 模块化单体、Gin 服务入口和统一配置加载。
- [x] Vue 3 + Vite + Element Plus + Pinia + vue-router 前端骨架。
- [x] 统一响应 `{code, message, data}`、错误码和分页约定。
- [x] SQLite 存储与迁移。
- [x] PostgreSQL 存储、方言迁移和驱动工厂。
- [x] Makefile、Dockerfile、CI、本地缓存目录和构建命令。
- [x] k3s namespace、RBAC、server、web、PostgreSQL、Ingress 清单。
- [-] 最新代码版本的服务器部署和 PostgreSQL 持久性统一复验。

### 阶段 1：P0 资源运维 MVP

- [-] 当前开发阶段，详见第 3 节。

### 阶段 2：P1 配置中心

- [ ] App / Environment / ConfigItem 数据模型与 Store。
- [ ] ConfigVersion 全量快照、版本差异和回滚。
- [ ] ConfigRelease 发布记录和审计。
- [ ] 渲染 ConfigMap/Secret 并 Apply。
- [ ] 为目标 Workload 写入配置版本 annotation 并触发受控滚动更新。
- [ ] Secret 配置项 AES-GCM 加密存储和脱敏展示。
- [ ] 配置中心前端页面与单元测试。

### 阶段 3：P1 应用发布

- [ ] Release 状态机、审批、暂停、恢复、失败和回滚。
- [ ] `Executor` 接口抽象。
- [ ] Tekton 执行器。
- [ ] 无 Tekton 时的受控 `kubectl-image` 降级执行器。
- [ ] 发布记录、审计、前端和 integration 测试。

### 阶段 4：P1 长期日志

- [ ] Fluent Bit 采集链路。
- [ ] Loki 单实例与 3~7 天保留策略。
- [ ] `logquery` Kubernetes/Loki source 切换。
- [ ] 跨 Pod、跨轮转、时间范围、上下文和下载。
- [ ] TraceId 查询和日志采样/截断。

### 阶段 5：P1 监控、事件与 AI

- [ ] metrics-server Node/Pod CPU、Memory 查询封装。
- [ ] 事件聚合、去重、规则告警和 webhook。
- [ ] CrashLoopBackOff、ImagePullBackOff、OOMKilled 等规则诊断。
- [ ] Pod/Deployment/日志/事件诊断上下文构建。
- [ ] OpenAI-Compatible LLM 客户端和结构化诊断报告。
- [ ] 诊断任务存储、历史记录和反馈。
- [ ] AI 修复建议到受控动作的映射。
- [ ] 所有 AI 修复动作经过 RBAC、二次确认和审计；禁止模型直接执行任意 Kubernetes 修改。
- [ ] 日志和 AI 上下文统一经过 sanitizer。

### 阶段 6：P2 增强能力

- [ ] 资产录入、授权和分类。
- [ ] SSH/RDP 远程连接、文件管理和录像。
- [ ] Pod WebSSH/exec 与命令审计。
- [ ] 端口转发可视化。
- [ ] 会话录像回放和行为审计强化。
- [R] 多集群正式启用。
- [R] 多节点调度策略辅助；指定 Pod 调度节点不在当前 P0 实施。

## 3. P0 功能 Checklist

### 3.1 认证、权限、存储和审计

- [x] 本地 email + bcrypt 密码登录。
- [x] JWT access/refresh 会话。
- [x] TOTP MFA 全局开关、首次绑定和登录验证。
- [x] TOTP Secret AES-GCM 加密存储和旧明文升级兼容。
- [x] 登录/MFA 失败进程内速率限制。
- [x] Admin 用户、角色、状态和他人 MFA 重置管理。
- [x] Viewer / Operator / ConfigAdmin / Auditor / Admin 五级 RBAC。
- [x] SQLite/PostgreSQL 双驱动和迁移。
- [x] 写请求审计与审计查询页面。
- [x] 审计请求体敏感字段 sanitizer。
- [ ] LDAP/OIDC；属于认证源扩展，不阻塞本地认证 P0。
- [ ] refresh token 服务端吊销、恢复码和 MFA 主密钥轮换；属于后续安全增强。

### 3.2 集群与概览

- [x] in-cluster 单集群标识和前端禁用切换入口。
- [x] Node、Namespace、Pod、异常 Pod 数量和 Node allocatable 概览。
- [-] CPU/Memory 实际使用率；归入 P1 metrics-server。
- [R] 多集群注册和切换；P2 实施。

### 3.3 Kubernetes 资源管理

#### Namespace

- [x] 列表、状态和创建时间。
- [x] 详情、Labels/Finalizers/Conditions、自身 Event、资源计数、有效 Pod requests/limits 和关联 Pod/Workload/Service/Ingress/PVC。
- [x] Namespace YAML 只读导出。
- [x] 删除保持关闭。

#### Node

- [x] 列表、Ready 状态、allocatable 和 YAML。
- [x] 详情、roles、Conditions、地址、污点、系统信息、Capacity/Allocatable、有效 Pod requests/limits 和关联 Pod/Workload/Event。
- [x] 资源声明占比明确标注为非实时使用率；实时指标留给 P1 metrics-server。
- [x] Node 写操作保持关闭。
- [R] 指定调度节点、NodeSelector/Affinity/Toleration 辅助能力。

#### Pod

- [x] 列表、Phase、Ready、状态原因、节点和重启次数。
- [x] 详情、网络/QoS/ServiceAccount、Init Container、Conditions。
- [x] 当前状态、最近终止状态、Exit Code、OOMKilled 上下文。
- [x] ReplicaSet→Deployment、Job→CronJob 控制器链。
- [x] Pod 关联 Event。
- [x] YAML 只读。
- [x] 显式删除和 `confirm=true`。
- [x] ReplicaSet/StatefulSet/DaemonSet Pod 删除重建式重启；Job Pod 和独立 Pod 返回 409。
- [-] 最新代码的真实控制器行为验收。

#### Deployment

- [x] 列表、详情、Conditions、发布策略和副本状态。
- [x] Deployment→ReplicaSet→Pod/Event 关联。
- [x] 从关联 Pod 下钻详情和历史/实时日志。
- [x] YAML 查看/编辑、扩缩容和滚动重启。
- [-] 最新代码的真实 k3s 验收。

#### StatefulSet

- [x] 列表和详情。
- [x] Headless Service、Revision、Pod 管理/更新策略和 PVC 模板。
- [x] YAML 查看/编辑、扩缩容和 RollingUpdate 重启。
- [x] OnDelete 策略返回 409。
- [-] 最新代码的真实 k3s 验收。

#### DaemonSet

- [x] 列表和详情。
- [x] Selector、NodeSelector、Tolerations 和调度状态。
- [x] YAML 查看/编辑和 RollingUpdate 重启。
- [x] OnDelete 策略返回 409；不提供扩缩容。
- [-] 最新代码的真实 k3s 验收。

#### ReplicaSet

- [x] 列表和 YAML 只读。
- [x] Pod 控制器链中可识别 ReplicaSet→Deployment。
- [x] 独立详情、所属 Deployment、关联 Pod/Event 和日志下钻。
- [x] 直接扩缩容/删除保持关闭，避免与 Deployment 管理冲突。
- [-] 最新代码的真实 k3s 验收。

#### Job

- [x] 列表和 YAML 查看/编辑。
- [x] 详情、执行策略、Conditions 和 CronJob owner。
- [x] 直属 Pod、Job/Pod Event 以及 Pod 详情和历史/实时日志下钻。
- [x] Job 删除和重新运行保持关闭；Job Pod 通用重启返回 409。
- [-] 最新代码的真实 k3s 验收。

#### CronJob

- [x] 列表和 YAML 查看/编辑。
- [x] 详情、时区/并发/错过调度/历史保留策略和 JobTemplate。
- [x] 直属历史 Job、后代 Pod、CronJob/Job/Pod Event 和日志下钻。
- [x] 暂停/恢复要求 Operator/Admin、`confirm=true`、前端二次确认和审计。
- [x] 立即运行和删除保持关闭；立即运行等待 jobs/create、唯一命名和幂等设计。
- [-] 最新代码的真实 k3s 验收。

#### Service / Ingress

- [x] Service 列表、类型、ClusterIP、Selector、端口和 YAML 编辑。
- [x] Service 详情、流量策略和结构化端口。
- [x] EndpointSlice 优先、传统 Endpoints 回退，Ready/Serving/Terminating 状态展示。
- [x] selector/targetRef 关联 Pod、Service/Endpoint/Pod Event，以及 Pod 详情和历史/实时日志下钻。
- [-] 最新代码的真实 k3s 验收。
- [x] Ingress 列表、Class、Host、Address、TLS 和 YAML 编辑。
- [x] Ingress 默认后端、Host/Path/PathType、TLS 元数据和 Resource backend 详情。
- [x] 后端 Service/端口存在性校验、Service 去重，以及 EndpointSlice/Endpoints/Pod/Event 和日志下钻。
- [-] 最新代码的真实 k3s 验收。
- [ ] 公网 IP `:80` 无 Host 访问规则；当前 Host-only Ingress 会由 Traefik 返回 404。

#### PV / PVC / StorageClass

- [x] 列表、容量、状态、StorageClass、Claim 和 YAML。
- [x] PVC/PV 双向详情、有效绑定校验、Pod/Workload、容器挂载/块设备路径和 Event 关联。
- [x] CSI 等卷源只展示安全摘要，不返回 attributes、volume handle 或 SecretRef。
- [-] 最新代码的真实 k3s 验收。
- [x] 删除保持关闭。

#### ConfigMap / Secret

- [x] ConfigMap 列表、Key 和 YAML 只读。
- [x] ConfigMap 直接写入保持关闭，等待配置中心版本发布链路。
- [ ] Secret 安全只读列表和受控明文读取。
- [x] Secret Kubernetes RBAC 默认关闭。
- [ ] Secret 写入；只能通过 P1 配置中心并受 Admin/ConfigAdmin 权限控制。

#### Event

- [x] Namespace Event 列表和 involved kind/name 过滤。
- [x] Pod、Deployment、ReplicaSet、Job、CronJob、Service、Ingress、PVC 和 PV 详情关联 Event。
- [ ] 独立事件页和更多资源详情联动。
- [ ] 事件聚合、告警和 webhook；归入 P1。

### 3.4 Pod 日志

- [x] Kubernetes API 当前容器日志。
- [x] `previous=true` 最近一次容器实例日志。
- [x] limit、from、keyword 和 level 过滤。
- [x] WebSocket `follow=true` 实时日志。
- [x] JWT 通过 WebSocket 子协议传递，不进入 URL。
- [x] REST/WebSocket 输出 sanitizer。
- [x] 前端历史/实时双模式、开始/停止和 5000 行上限。
- [-] 真实 k3s 日志、previous 和 WebSocket 验收。
- [ ] 跨 Pod/轮转长期历史；归入 P1 Loki。

### 3.5 资源写操作安全

- [x] 写接口仅 Operator/Admin。
- [x] 所有写请求经过审计中间件。
- [x] 删除类接口前端确认 + 后端 `confirm=true`。
- [x] YAML 更新仅开放 Deployment/StatefulSet/DaemonSet/Job/CronJob/Service/Ingress。
- [x] YAML 保存和重启分为两个请求、两条审计记录。
- [x] `pods/exec`、Secret 写、Namespace 删除和 Node 写保持关闭。

### 3.6 缓存、测试与部署

- [ ] informer/cache 和资源关联 mapper；当前详情接口仍可能按 Namespace 直查多个资源。
- [x] 资源/Workload fake client 单测。
- [x] k3d integration 测试框架与控制器行为测试代码。
- [x] `make test`、`make lint`、`make build` 本地门禁。
- [x] OpenAPI、Kubernetes YAML 和 Shell 语法检查。
- [-] Docker 镜像构建；因网络状况后置。
- [-] k3d 测试实际运行；当前只编译 integration 测试代码。
- [-] 最新代码部署到服务器并完成真实 k3s 验收。
- [ ] 公网 Ingress 无 Host/IP 访问方案实施和验收。

## 4. 当前固定开发顺序

每次只完成一个明确任务，完成 API、后端、前端、测试和文档闭环后再进入下一项。

1. [x] **Job 详情闭环**：状态/Conditions、Pod/Event/日志关联和安全操作边界已完成，等待真实 k3s 验收。
2. [x] **CronJob 详情闭环**：调度策略、历史 Job/Pod/Event、暂停/恢复和安全边界已完成，等待真实 k3s 验收。
3. [x] **网络关联代码闭环**：Service→EndpointSlice/Endpoints→Pod 和 Ingress→Service→EndpointSlice/Endpoints→Pod/Event 已完成，等待真实 k3s 验收。
4. [x] **存储关联代码闭环**：PVC/PV→Pod/Workload、容器挂载路径、双向绑定和 Event 已完成，等待真实 k3s 验收。
5. [x] **Namespace/Node 详情代码闭环**：Conditions、污点、地址、资源声明统计、Pod/Workload/网络/存储关联和只读边界已完成，等待真实 k3s 验收。
6. **Secret 安全只读**：下一项先设计角色、脱敏和审计边界，再开放最小 Kubernetes 权限。
7. **informer/cache + resource mapper**：替换重复全量直查，保持现有 API 契约。
8. **统一验收**：本地门禁、Docker、k3d、服务器、真实 k3s、PostgreSQL 持久性和 Pod 日志。
9. **Ingress 公网入口**：单独实施无 Host/IP 规则或明确域名访问方案。
10. **进入 P1**：配置中心 → 发布执行器 → Loki → metrics/event 规则 → AI 规则诊断与受控修复。

## 5. 验收原则和已知边界

- P0 完成标准不是“有列表接口”，而是 API、后端、前端、RBAC/审计、测试和真实集群行为形成闭环。
- 本地测试通过不等同于真实 k3s 已验收；代码闭环但未部署的能力标记为“待验收”。
- Kubernetes API 日志不等于长期日志库，Pod 删除或日志轮转后不能保证保留。
- Kubernetes 没有 Pod 原地重启 API，平台重启语义是删除 ReplicaSet/StatefulSet/DaemonSet 管理的 Pod 后由控制器重建；Job Pod 不适用该语义并返回 409。
- Deployment/StatefulSet/DaemonSet YAML 中 Pod template 变化会由 Kubernetes 自动 rollout，不应无条件额外重启。
- StatefulSet/DaemonSet `OnDelete` 不会自动滚动替换 Pod，平台保持 409 安全边界。
- CronJob 暂停只阻止未来调度，不停止已有 Job；恢复可能按 `startingDeadlineSeconds` 处理错过的调度。
- 当前 Ingress 只匹配配置的 Host；直接使用公网 IP `:80` 会命中 Traefik 默认 404。
- 不把真实服务器凭据、GitHub Token、JWT/TOTP/PostgreSQL 密钥写入代码、文档、脚本或 Git 历史。
- 不引入微服务、Redis、消息队列、ELK/OpenSearch、Prometheus 全家桶或服务网格。

## 6. 最近一次本地验证基线

以下检查在 2026-08-13 已通过：

- [x] `make test`
- [x] `make lint`
- [x] `make build`
- [x] integration 测试代码编译（未创建 k3d）
- [x] OpenAPI YAML 解析
- [x] k3s/test YAML 解析
- [x] Shell `bash -n`
- [x] `git diff --check`

Vite 仍有主包超过 500 kB 的非阻塞警告；不在当前 P0 核心资源任务中顺带重构。
