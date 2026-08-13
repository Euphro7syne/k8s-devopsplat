# P0 资源管理

资源管理模块通过 Kubernetes API 提供列表、详情、YAML 查看和受控写操作。读取接口面向 `viewer/operator/configadmin/auditor/admin`，写接口仅面向 `operator/admin`，所有 POST/PUT/DELETE 请求统一经过审计中间件。

P0 各资源的实际完成度与后续顺序统一维护在 [p0-status.md](p0-status.md)，不能再把“已有列表接口”等同于“完整闭环”。

## Pod 诊断上下文

Pod 详情用于日常排障，也作为后续规则诊断和 LLM 报告的数据基础，当前包含：

- Phase、Ready、节点、Pod IP/Host IP、QoS、ServiceAccount、重启策略和启动时间。
- 容器当前状态、等待/退出原因、Exit Code、重启次数，以及最近一次终止状态（如 `OOMKilled` / exit 137）。
- Pod OwnerReferences 和解析后的控制器链，例如 `ReplicaSet → Deployment`、`Job → CronJob`。
- Pod Conditions、Labels 和按 `involvedObject=Pod/<name>` 筛选的关联 Event。

指定 Pod 调度节点本轮不开放写操作；NodeSelector、Affinity、Tolerations 等调度策略保留为后续多节点 k8s 能力。

## Namespace / Node 只读诊断详情

Namespace 详情用于快速判断一个命名空间内资源是否完整、Pod 是否异常以及声明资源规模，当前包含：

- 状态、Labels、Finalizers、Conditions 和 Namespace 自身 Event；Event 按名称与 UID 过滤，不把 Namespace 内所有资源事件混入其中。
- Pod 总数、Ready/异常 Pod，以及 Deployment、StatefulSet、DaemonSet、ReplicaSet、Job、CronJob、Service、Ingress、PVC 和 ConfigMap 数量。
- 非终态 Pod 的有效 requests/limits 汇总：普通容器求和，initContainers 按每种资源取最大值，再加 Pod overhead；Succeeded/Failed Pod 不进入资源声明统计。
- Pod、顶层 Workload、Service、Ingress 和 PVC 下钻；ReplicaSet 继续解析到 Deployment，Job 继续解析到 CronJob，并按 kind/namespace/name 去重。
- Namespace YAML 只读导出；删除继续保持关闭。

Node 详情用于查看节点健康、可调度资源和承载工作负载，当前包含：

- Ready 状态、roles、unschedulable、PodCIDRs、Labels、Addresses、Taints、Conditions 和系统/运行时版本。
- Capacity、Allocatable、非终态 Pod 的有效 requests/limits，以及 CPU/Memory requests 和 Pod 数相对 Allocatable 的百分比。
- 调度到该 Node 的全部 Pod、顶层 Workload 和 Node 自身 Event；Event 按名称与 UID 过滤。

上述 CPU/Memory、EphemeralStorage 和 Pod 数均为 Kubernetes 声明值或容量值，不是实时使用率。实时使用率继续由 P1 metrics-server 提供。Node cordon/drain、Taint 修改和指定 Pod 节点调度都不在本项开放，后续多节点 k8s 能力只保留扩展位。

## Pod 日志

P0 日志能力分为：

- 历史查看：调用 Kubernetes Pod Logs API，支持当前容器、最近一次容器实例（`previous=true`）、尾部行数、起始时间、关键字和级别筛选。
- 实时跟随：通过 `/ws/v1/.../logs/follow` WebSocket 转发 Kubernetes `follow=true` 日志，体验等价于 `kubectl logs -f` / `tail -f`，支持开始和停止。

Kubernetes API 日志不是长期日志库，只能读取节点仍保留的当前 Pod 日志和最近一次容器实例。Pod 删除、节点日志轮转或重建后不保证还能查询；跨 Pod、跨轮转、按天检索的长期历史由 P1 Fluent Bit + Loki 提供。

REST 和 WebSocket 日志行在返回前统一经过现有 `sanitizer`。WebSocket JWT 使用 `Sec-WebSocket-Protocol` 携带，不放入 URL query，避免进入代理访问日志。

## Deployment 诊断详情

Deployment 详情用于从工作负载直接下钻到故障 Pod，当前包含：

- 副本状态、Generation/ObservedGeneration、暂停状态、RollingUpdate 策略、MaxSurge/MaxUnavailable、ProgressDeadline 和 RevisionHistoryLimit。
- Deployment Conditions、Selector、容器镜像。
- 通过 `OwnerReferences` 识别的直属 ReplicaSet，包含 revision 和副本状态。
- 通过 ReplicaSet `OwnerReferences` 继续识别的关联 Pod，可直接打开 Pod 诊断详情与历史/实时日志。
- 汇总 Deployment、直属 ReplicaSet 和关联 Pod 的 Event，并按最后发生时间倒序展示。

当前实现按 Namespace 读取 ReplicaSet、Pod 和 Event 后建立关联，符合 P0 单节点、小规模集群假设。后续第 7 项会通过 informer/cache 和资源关联 mapper 减少重复直查，但不会改变 API 契约。

## ReplicaSet 诊断详情

ReplicaSet 详情用于定位 Deployment rollout 中间层的问题，当前包含：

- Revision、ObservedGeneration、期望/当前/就绪/可用/完全匹配/不可用副本。
- Selector、镜像、MinReadySeconds 和 ReplicaSet Conditions。
- 通过 `OwnerReferences` 识别所属 Deployment，并可继续打开 Deployment 详情。
- 通过 ReplicaSet UID 识别直属 Pod，可继续打开 Pod 诊断详情与历史/实时日志。
- 汇总 ReplicaSet 和直属 Pod 的 Event，并排除同名旧 UID 资源遗留的 Event。

ReplicaSet 通常由 Deployment 控制，平台不开放直接扩缩容或删除，避免用户操作与 Deployment controller 冲突。独立 ReplicaSet 当前同样只提供只读诊断和 YAML。

## Job 诊断详情

Job 详情用于判断一次性任务是仍在执行、已经完成，还是因重试耗尽等原因失败，当前包含：

- Parallelism、Completions、BackoffLimit、ActiveDeadlineSeconds、完成后 TTL、CompletionMode、Suspend 和 ManualSelector。
- Selector、镜像、开始/完成时间、成功/失败/运行中数量和 Job Conditions。
- 通过 `OwnerReferences` 识别所属 CronJob；手动创建的 Job 显示为无 CronJob owner。
- 通过 Job UID 识别直属 Pod，可继续打开 Pod 详情以及当前/上一次容器日志和实时日志。
- 汇总 Job 和直属 Pod 的 Event，并排除同名旧 UID 资源遗留的 Event。

平台当前不开放 Job 删除或重新运行。Kubernetes Job 不能通过删除原 Pod 可靠“重启”：删除运行中 Pod 可能影响失败计数和 backoff，删除已完成 Pod 也不会重新执行 Job。因此 Job 管理的 Pod 调用通用 restart API 会返回 HTTP 409；未来若实现“重新运行”，必须从受控模板创建一个新名称的 Job，并单独落实 RBAC、二次确认和审计。

## CronJob 诊断与调度控制

CronJob 详情用于检查定时任务的调度配置、并发行为和保留的执行历史，当前包含：

- Schedule、TimeZone、ConcurrencyPolicy、StartingDeadlineSeconds 和 Suspend。
- Successful/FailedJobsHistoryLimit、上次调度时间和上次成功时间。
- JobTemplate 的 Parallelism、Completions、BackoffLimit、ActiveDeadlineSeconds、完成后 TTL、CompletionMode、RestartPolicy 和镜像。
- 通过 CronJob UID 识别直属历史 Job；每个 Job 保留 Conditions、直属 Pod、Job/Pod Event，可继续下钻 Pod 详情以及历史/实时日志。
- 汇总 CronJob、直属 Job 和后代 Pod 的 Event，并排除同名旧 UID 资源遗留的 Event。

Operator/Admin 可通过独立 POST 接口暂停或恢复未来调度，两种操作都要求 `confirm=true` 并经过审计。暂停不会终止已经开始的 Job；恢复后，Kubernetes 是否补偿暂停期间错过的调度取决于 `startingDeadlineSeconds` 等控制器规则，因此前端必须在确认框中提示该行为。

“立即运行”和删除继续保持关闭。立即运行并不是重启 CronJob，而是从 JobTemplate 创建一个新名称的 Job，需要新增 `jobs/create` 最小权限、唯一命名、幂等、二次确认和审计设计；删除属于高风险删除操作，当前 P0 不开放。

## Service 网络关联详情

Service 详情用于确认流量入口最终落到哪些 Pod，并区分“selector 已选中”和“已经进入可服务端点”两种状态，当前包含：

- Service 类型、ClusterIPs、ExternalName、IP Families/Policy、SessionAffinity、内外部流量策略、PublishNotReadyAddresses 和 LoadBalancerSourceRanges。
- 结构化 Service Port：name、protocol、port、targetPort、nodePort 和 appProtocol。
- 优先读取 `discovery.k8s.io/v1 EndpointSlice`，展示地址、端口、Ready、Serving、Terminating、Node、Zone 和 targetRef；仅在没有 EndpointSlice 时回退同名传统 Endpoints，避免重复显示相同后端。
- Pod 关联同时包含 Service selector 命中的 Pod，以及 Endpoint targetRef 指向的 Pod。因此尚未进入 ready endpoint 的 selector Pod仍会展示，便于定位 readiness 或端点同步问题。
- 汇总 Service、当前采用的 EndpointSlice/Endpoints 和关联 Pod Event，并按对象 UID 排除同名旧资源遗留事件。
- 前端可从关联 Pod 继续打开诊断详情、当前/上一次容器日志和实时日志。

ExternalName、无 selector 或手工维护 Endpoints 的 Service 不一定能关联 Pod；此时平台仅依据 endpoint targetRef 建立关联，并在页面明确提示边界。Service 详情是只读诊断能力，本轮不新增删除或其他高风险写操作。

## Ingress 网络关联详情

Ingress 详情用于从入口规则一路确认到最终 Pod，当前包含：

- IngressClass、LoadBalancer Address、Host、创建时间，以及 TLS Host/Secret 元数据。Secret 内容不会读取或展示。
- 默认后端和每条 HTTP Host/Path/PathType 规则，支持 Service backend 的命名端口和数字端口，也保留 Resource backend 的 APIGroup/Kind/Name。
- 对 Service backend 同时校验 Service 是否存在、Ingress 引用的 Service Port 是否存在；缺失对象仍保留在规则表中用于诊断，不会因为关联失败而静默丢弃。
- 对重复引用的后端 Service 去重，并复用 Service 详情能力继续展示 EndpointSlice/Endpoints、Ready/Serving/Terminating、selector/targetRef Pod 和 Service 事件。
- 汇总 Ingress、关联 Service、EndpointSlice/Endpoints 和 Pod Event，并按对象 UID 排除同名旧资源遗留事件。
- 前端可从后端 Service 打开完整 Service 详情，也可从后端 Pod 下钻诊断详情和历史/实时日志。

本详情接口只读，不新增 Ingress 删除或其他高风险写操作。公网 IP `:80` 的无 Host 访问规则属于部署入口策略，继续作为独立任务实施；当前 Host-only Ingress 不会自动匹配直接 IP 请求。

## PVC/PV 存储关联详情

PVC/PV 详情用于回答“哪个工作负载正在使用这块存储、具体挂到了哪个容器路径、绑定关系是否仍然有效”，当前包含：

- PVC 状态、请求/实际容量、AccessModes、VolumeMode、StorageClass、Selector、DataSource 和 Conditions。
- PVC→PV 与 PV→PVC 双向下钻；PVC→PV 要求 `spec.volumeName` 指向目标 PV 并校验 claimRef，PV→PVC 依据 claimRef namespace/name/UID 读取目标 Claim，且 PVC 已有 volumeName 时必须与 PV 一致，避免同名旧 Claim 被错误关联。
- 通过 Pod `spec.volumes[].persistentVolumeClaim.claimName` 识别实际使用者，再匹配普通容器、initContainer 和 ephemeralContainer 的 `volumeMounts` / `volumeDevices`，展示 volume 名、挂载路径、subPath、设备路径和只读状态。
- 复用 Pod controller chain，将 ReplicaSet 解析到 Deployment、Job 解析到 CronJob，并直接保留 StatefulSet/DaemonSet 等顶层 Workload；前端可继续下钻 Workload、Pod 详情以及历史/实时日志。
- 汇总 PVC、有效绑定 PV、关联 Pod 和顶层 Workload 的 Event，并按 UID 排除同名旧对象遗留事件。
- PV 展示 MountOptions、NodeAffinity、VolumeMode 和卷源安全摘要。CSI 只返回 driver、fsType、readOnly，不返回 `volumeAttributes`、volume handle 或任何 SecretRef；其他卷源同样只返回诊断必需的非敏感字段。

该能力完全只读。PVC/PV 删除、PVC 扩容和 StorageClass 修改继续关闭；后续若开放必须单独设计 RBAC、确认、Kubernetes 行为边界和审计。

## StatefulSet 闭环

StatefulSet 当前支持：

- 分页列表。
- 详情：副本状态、Headless Service、Pod 管理策略、更新策略、Revision、Selector、镜像和 VolumeClaimTemplates。
- YAML 查看与受限编辑。
- 0~100 副本扩缩容。缩容不会删除 StatefulSet 创建的 PVC。
- 滚动重启：更新 Pod template 的 `ops.platform/restarted-at` annotation，由 Kubernetes `RollingUpdate` 策略依次重建 Pod。

使用 `OnDelete` 更新策略的 StatefulSet 不会因 Pod template 更新自动重建 Pod，因此平台拒绝直接执行“滚动重启”，并提示操作者逐个重启受控 Pod。

## DaemonSet 闭环

DaemonSet 当前支持：

- 分页列表。
- 详情：期望/当前/就绪/已更新/可用/不可用/误调度数量、更新策略、Selector、NodeSelector、Tolerations 和镜像。
- YAML 查看与受限编辑。
- 滚动重启：更新 Pod template 的 `ops.platform/restarted-at` annotation，由 Kubernetes `RollingUpdate` 策略逐节点重建 Pod。

DaemonSet 的副本数由符合调度条件的节点数量决定，平台不提供扩缩容操作。使用 `OnDelete` 更新策略时，平台返回 HTTP 409，提示操作者逐个重启 DaemonSet 管理的 Pod。

## YAML 配置与重启解耦

YAML 保存和工作负载重启是两个独立审计操作。保存 Deployment/StatefulSet/DaemonSet YAML 后，前端会说明配置生效规则，并让操作者选择是否立即滚动重启，后端不会在 YAML 更新时隐式触发额外重启。

- `spec.template` 发生变更时，Kubernetes 会自动触发新一轮 rollout，通常不需要再次重启。
- ConfigMap/Secret 以环境变量方式注入，或应用只在启动时读取挂载文件时，配置变更后需要重启 Pod 才能生效。
- 选择“立即重启”会调用独立的 Deployment/StatefulSet/DaemonSet restart API，因此 YAML 保存与重启会生成两条审计记录。

## Pod 重启语义

Kubernetes 不提供 Pod 原地重启操作。平台的：

```text
POST /api/v1/namespaces/{namespace}/pods/{pod}/restart?confirm=true
```

会执行以下安全检查：

1. 后端强制要求 `confirm=true`。
2. 读取 Pod 的 controller owner reference。
3. 只允许 ReplicaSet、StatefulSet 或 DaemonSet 管理的 Pod；Job 管理的 Pod 明确返回 HTTP 409。
4. 删除目标 Pod，由对应控制器创建替代 Pod。

Deployment 的 Pod 直接由 ReplicaSet 管理，因此属于允许重启的受控 Pod。Job Pod、独立 Pod、静态 Pod 或无法识别控制器的 Pod 会返回 HTTP 409，不会被删除。显式删除 Pod 仍是单独的 DELETE 操作，需要二次确认，不能和“重启”语义混用。

Pod 重启使用独立 POST 路由，审计记录能够和显式 DELETE Pod 区分；审计资源路径、namespace、操作者、来源 IP 和时间都会写入 `audit_logs`。

## 真实集群集成测试

资源写操作除了 fake client 单测，还通过临时 k3d 集群验证 Kubernetes 控制器的真实行为：

- StatefulSet 详情、扩缩容和滚动重启。
- StatefulSet 管理 Pod 删除后的同名 Pod 重建与 UID 变化。
- Deployment/ReplicaSet 管理 Pod 删除后的替代 Pod 创建。
- Job 详情的 Condition、直属 Pod/Event 关联，以及 Job Pod 重启返回 HTTP 409 且不会被删除。
- CronJob 调度详情、历史 Job/Pod/Event 关联，以及暂停/恢复状态和审计记录。
- Service 详情的 EndpointSlice 优先、传统 Endpoints 回退、selector/targetRef Pod 和 Event 关联。
- Ingress 规则、后端 Service/端口校验，以及 Service→EndpointSlice/Endpoints→Pod/Event 关联。
- StatefulSet PVC→Pod/StatefulSet/挂载路径关联，以及 PV→PVC/Pod/Workload 反查。
- Namespace 资源计数、有效 Pod requests/limits、关联资源与顶层 Workload；Node Capacity/Allocatable/Requests、Pod/Workload 与健康详情。
- 独立 Pod 重启返回 HTTP 409，且不会被删除。
- StatefulSet scale/restart 与 Pod restart 的审计记录落入 SQLite。
- DaemonSet 详情、滚动重启、OnDelete 冲突和审计记录。

运行 `make test-integration` 会创建名称唯一的临时 k3d 集群、本机启动一个使用临时 SQLite 的 `ops-server`，测试结束后自动清理。该目标不会读取生产 kubeconfig、连接生产 PostgreSQL，也不会在普通 `make test` 中启动 Docker；普通测试只编译集成测试代码，避免接口变更导致测试长期失效。
