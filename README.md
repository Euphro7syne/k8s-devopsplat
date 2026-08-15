# K8s Ops Platform（k3s 运维平台）

> 面向**单节点 k3s** 的一体化运维平台：资源管理、配置中心、日志查询、应用发布、审计与 AI 诊断。
> 定位是「**轻量、模块化、可演进**的模块化单体」，不是大型分布式运维系统。

---

## 1. 项目简介

针对生产环境为单节点 k3s（约 **65 个业务 Pod、4~6 个 Namespace**）的场景，构建一个运行在集群内的运维控制台，替代日常 `kubectl` 手工操作，把「看资源、改配置、查日志、发版本、审行为」收敛到一个入口。

### 1.1 设计原则（来自参考文档，已确认）

1. **不做一开始就做重**：不做多副本 HA、Kafka/RabbitMQ、ELK/ClickHouse、服务网格、过度微服务化——单节点下平台自身的资源开销必须远小于业务。
2. **模块化单体**：一个后端主服务（`ops-server`）+ 一个前端控制台（`ops-web`）+ 一个轻量日志采集器（Fluent Bit），AI 先作为后端内部模块，后续再拆。
3. **AI 是能力不是依赖**：AI 诊断不影响主流程，先规则引擎、后 LLM。
4. **配置解耦**：应用镜像、应用配置、环境配置、发布记录分离；平台不直接到处改 ConfigMap。
5. **审计贯穿始终**：操作审计、会话录像、Pod 登录审计从第一天就内置，不做事后补。

### 1.2 规模假设（写死进容量设计）

| 维度 | 假设 | 影响 |
|---|---|---|
| 节点 | 单节点 k3s | 平台部署 1 副本即可，不需要 HA |
| Pod 数 | ~65 | 列表分页、Informer 缓存压力极小 |
| Namespace | 4~6 个 | 不做多租户，只做角色分级 |
| 并发用户 | 运维/开发若干 | 不引入 Redis 等外部缓存；Kubernetes 只读资源使用进程内 Informer，本地可用 SQLite，k3s 正式部署使用 PostgreSQL |
| 日志量 | 中小 | Loki 单实例，保留 3~7 天 |

---

## 2. 功能范围与分期

功能清单按 P0 / P1 / P2 排期，**P0 是可交付的 MVP 闭环**，P1/P2 是演进方向：

### P0 — 资源运维 MVP（第 1 阶段，3~6 周）

- [ ] 登录认证：LDAP / email 登录、JWT 会话、MFA（TOTP）开关
- [ ] 权限管理：角色分级（Viewer / Operator / ConfigAdmin / Auditor / Admin）
- [ ] 集群概览：节点状态、CPU/Memory、Pod 数、异常 Pod、资源统计
- [ ] 资源管理：Namespace、Node、Pod、Deployment、StatefulSet、ReplicaSet、DaemonSet、Job、CronJob、Service、Ingress、PV/PVC/StorageClass、ConfigMap/Secret、Event
- [ ] 资源操作：查看/编辑 YAML、删除 Pod、重启/扩缩容工作负载、查看事件
- [ ] Pod 日志：通过 Kubernetes API 直查（当前/上一次容器日志）
- [ ] 操作审计：所有写操作的审计落库

### P1 — 配置 / 发布 / 日志 / AI（第 2~4 阶段，约 12~20 周）

- [ ] 配置中心：App / Environment / ConfigItem / ConfigVersion / ConfigRelease，发布、对比、回滚、审计
- [ ] 应用发布：Tekton 流水线 + 发布审批 + 回退 / 暂停
- [ ] 日志平台：Fluent Bit → Loki，按 namespace/pod/container/时间/关键字/级别查询、上下文、下载
- [ ] 容器监控：基于 metrics-server 的 CPU/Memory 展示（不引入 Prometheus 全家桶）
- [ ] 事件中心：事件聚合 + 规则告警（规则引擎，webhook 通知）
- [ ] AI 诊断：先规则诊断（CrashLoopBackOff / ImagePullBackOff / OOMKilled 等），后接 LLM 生成报告

### P2 — 资产 / 远程 / 深度审计（增强阶段）

- [ ] 资产管理：资产录入、授权、分类
- [ ] 远程连接：SSH / RDP、屏幕录像、文件管理（基于 Guacamole 或自研轻量网关）
- [ ] WebSSH（Pod exec 终端）、Pod 登录审计
- [ ] 运维工具：端口转发（端口映射可视化操作）
- [ ] 行为审计强化：会话录像回放、登录/操作行为分析
- [ ] 多集群管理的正式启用（后端抽象已预留）

---

## 3. 总体架构

```text
                          ┌──────────────────────────┐
                          │      运维/开发人员        │
                          └────────────┬─────────────┘
                                       │ HTTPS
                                       ▼
                        ┌──────────────────────────────┐
                        │          Ops Web             │
                        │  Vue 3 + Vite + Element Plus │
                        │  资源/配置/日志/发布/资产/AI  │
                        └────────────┬─────────────────┘
                                     │ REST / WebSocket
                                     ▼
┌───────────────────────────────────────────────────────────────┐
│                       Ops Server（Go 模块化单体）              │
│                                                               │
│  Auth/RBAC ─ Authn(LDAP/OIDC/email) ─ MFA ─ JWT ─ RBAC       │
│  Resources ─ 资源查询/操作/事件（client-go + Informer）         │
│  Config    ─ 配置中心：版本、发布、回滚、同步 ConfigMap/Secret │
│  Release   ─ Tekton 流水线编排 + 审批 + 回退/暂停              │
│  LogQuery  ─ K8s API 直查 / Loki 查询（source 可切换）         │
│  Monitor   ─ metrics-server 数据封装                          │
│  AI        ─ 规则诊断 → LLM 报告（内部模块，可插拔）            │
│  Asset     ─ 资产授权 + 远程会话网关（SSH/RDP/录像）           │
│  Audit     ─ 操作审计 / 会话录像 / 行为审计（横切中间件）       │
└──────┬──────────────┬────────────────┬───────────────┬───────┘
       │              │                │               │
       ▼              ▼                ▼               ▼
┌─────────────┐ ┌────────────┐ ┌──────────────┐ ┌──────────────┐
│   k3s API   │ │ SQLite/PG  │ │ Fluent Bit→  │ │  Tekton      │
│ in-cluster  │ │ 平台数据    │ │ Loki 日志     │ │  (P1 可选)    │
│ ServiceAccount│ │ 配置版本/审计 │ │ (P1 可选)     │ │  (P1 可选)    │
└─────────────┘ └────────────┘ └──────────────┘ └──────────────┘
```

### 3.1 为什么是「模块化单体」

- 规模（1 节点 / 65 pod / 4~6 ns）下，拆分微服务只会增加 3~4 个 Deployment 和跨服务调用，纯增负担。
- 单二进制部署、升级、备份都简单；模块按 `internal/<module>` 内聚，未来真的需要拆（如 AI 服务）时，模块边界已经存在。

---

## 4. 技术栈选型（已定）

| 层 | 选型 | 理由 / 备注 |
|---|---|---|
| 后端 | **Go + Gin** + client-go + informer + gorilla/websocket | k8s 生态最成熟、单二进制、资源占用低（见参考文档） |
| 前端 | **Vue 3 + Vite + Element Plus**（已确认） | 中后台生态成熟，配合 Pinia、vue-router、xterm.js |
| 存储 | **SQLite + PostgreSQL** | `internal/store` 接口隔离；本地默认 SQLite，k3s 正式部署默认 PostgreSQL |
| 日志采集 | **Fluent Bit**（DaemonSet） | 比 Fluentd 轻；日志量小时可先跳过，直接 K8s API 查 |
| 日志存储 | **Loki** 单实例（P1） | 保留 3~7 天，限制资源、行截断 |
| 监控数据 | **metrics-server**（P1） | 不做 Prometheus 全家桶，先满足 CPU/Memory 展示 |
| 流水线 | **Tekton**（P1，可选组件） | 与 k8s 同源 CRD 体系，k3s 单节点可跑，需限制资源 |
| 远程连接 | Guacamole 或自研轻量 SSH/RDP 网关（P2） | 较重，放最后；WebSSH（Pod exec）可提前 |
| AI | OpenAI-Compatible API / 本地 Ollama（P2 开放） | 默认调用外部 API 并做脱敏；先规则后 LLM |
| 部署 | YAML（k3s 目录）→ Helm Chart 可演进 | MVP 直接 kubectl apply |

### 4.1 明确不引入（至少 MVP 阶段）

Kafka / RabbitMQ、ClickHouse / OpenSearch、ELK、Prometheus + Alertmanager 全家桶、istio / 服务网格、多副本 HA 控制面——单节点场景下这些组件本身比业务还重。

---

## 5. 关键架构决策与取舍（审阅结论）

| # | 决策 | 结论 |
|---|---|---|
| 1 | 多集群管理（功能清单有，参考文档建议不做） | **预留抽象，MVP 单集群**（已确认）：后端定义 `Cluster` 模型 + 注册表 + 按 cluster 上下文的 client 工厂；MVP 默认加载 in-cluster 集群，UI 预留切换入口，P2 再启用 |
| 2 | 审计（操作审计 / 录像 / Pod 登录审计 / 行为审计） | 参考文档弱化，功能清单强调 → **审计从 P0 就作为横切中间件内置**：所有写请求记 `audit_logs`，WebSSH 会话流式录像存储，P2 做回放与分析 |
| 3 | 资产远程（RDP/SSH/录像/文件） | 这是堡垒机/跳板机能力，最重 → 独立 `asset` 模块，P2 引入；先落地轻量的 WebSSH（Pod exec）与端口转发 |
| 4 | 容器监控 | 功能清单要求 → 用 metrics-server 数据 + 自研查询接口，避免一开始上 Prometheus；P2 若需要告警再评估 |
| 5 | 事件中心 + 告警 | 先做事件聚合展示 + 内置规则引擎（P1），webhook 通知；不做独立 alertmanager |
| 6 | 应用发布（Tekton + 审批 + 回退/暂停） | 发布引擎抽象为 `Release` 状态机（pending→approving→running→paused→succeeded/failed/rolled_back），Tekton 作为可选执行后端，避免与 Tekton 强耦合 |
| 7 | 配置中心 | 完全按参考文档的 App/Env/Item/Version/Release 模型；发布 = 渲染 ConfigMap/Secret → Apply → 打 annotation 触发滚动更新 |
| 8 | RBAC | 平台内部五级角色，与 k8s RBAC 分离：平台权限落平台数据库，写 k8s 用平台自身的 ServiceAccount（ClusterRole 最小授权） |
| 9 | 认证 | LDAP / email 登录 + JWT；MFA 用 TOTP（无额外组件）；企业微信登录走 OIDC provider 抽象（P1） |
| 10 | 存储 | SQLite 与 PostgreSQL 双实现；配置快照、审计、诊断记录全落库；k3s 默认使用 PostgreSQL StatefulSet |

### 5.1 安全红线

- 平台 ServiceAccount 使用**最小化 ClusterRole**（默认只读 + 明确的操作动词），`pods/exec`、`secrets` 写权限单独控制。
- 日志与 AI 上下文**必须脱敏**：password / token / secret / authorization / api_key / private_key 等一律打码。
- Secret 在平台内**加密存储**（AES-GCM，密钥来自挂载的 Secret），列表脱敏展示。
- 所有写操作记录 `user / action / resource / namespace / request_body / ip / time`，审计数据只追加不可改。

---

## 6. 核心模块设计

### 6.1 认证与权限（`internal/auth`）

- 登录：LDAP 绑定验证 / 本地 email + 密码；成功签发 JWT（短时效 + refresh）。
- MFA：TOTP（RFC 6238）；`auth.mfa_enabled=true` 时全员强制动态码并在首次登录完成绑定，`false` 时登录只校验账号密码；Admin 可为丢失身份验证器的其他用户重置绑定。
- RBAC 角色与权限矩阵：

| 能力 | Viewer | Operator | ConfigAdmin | Auditor | Admin |
|---|:-:|:-:|:-:|:-:|:-:|
| 查看资源 / 日志 | ✅ | ✅ | ✅ | ✅ | ✅ |
| Secret 元数据 / key | | | ✅ | | ✅ |
| Secret 单 key 明文（确认+审计） | | | | | ✅ |
| 删除 Pod、扩缩容、重启 | | ✅ | | | ✅ |
| 配置中心管理 | | | ✅ | | ✅ |
| 审计 / 会话录像查看 | | | | ✅ | ✅ |
| 资产 / 用户 / 权限管理 | | | | | ✅ |

### 6.2 资源管理（`internal/resources` + `internal/workload`）

- 统一封装：list / get / create / update / delete / patch / yaml 导出。
- 当前详情接口已建立 Namespace/Node→Pod/顶层 Workload/关联资源、Pod↔Workload、Service↔Endpoint↔Pod、Ingress↔Service↔Pod 和 PVC↔PV↔Pod↔Workload 的只读关联；Namespace/Node 的 CPU/Memory/Pod 数据为 Capacity/Allocatable 和 Pod requests/limits 声明值，实时使用率留给 P1 metrics-server。
- P0 非敏感只读资源通过进程内 typed Informer 缓存和 `resource_mapper` 解析关联；缓存未启用或未同步时自动回退 Kubernetes API，YAML/写操作继续直连。Secret 明确不进入共享 Informer，避免明文长期驻留缓存。
- k3s 默认 Ingress 使用无 Host 规则，支持通过域名或公网 IP 的 80 端口访问；生产可收敛为固定域名并配置 TLS。
- 资源页提供 Namespace Event 独立视图和 involved kind/name 筛选；事件聚合与告警留 P1。
- PVC→PV 校验 volumeName 与 claimRef，PV→PVC 校验 claimRef namespace/name/UID 并拒绝冲突的 PVC volumeName；容器挂载覆盖普通/init/ephemeral container 的 mount/device，CSI 等卷源只返回非敏感摘要。
- Secret 列表/详情仅向 ConfigAdmin/Admin 返回类型、labels、key 名与大小；Admin 可经 `confirm=true` 的独立 POST 审计接口读取单个 key。Secret 不进入通用 YAML，写入留给 P1 配置中心。
- 写操作（delete/scale/restart/edit）统一走 `internal/workload`，并在中间件层记审计。

### 6.3 配置中心（`internal/configcenter`）

模型：`App` → `Environment` → `ConfigItem`(key/value/is_secret) → `ConfigVersion`(快照) → `ConfigRelease`。

发布链路：

```text
配置中心 → 生成 ConfigVersion（全量快照）
       → 渲染 ConfigMap / Secret
       → Apply 到目标 Namespace
       → 给 Deployment/StatefulSet 打 annotation
         (ops.platform/config-version, ops.platform/released-at)
       → 触发滚动更新
```

支持：配置项编辑（Secret 脱敏）、版本对比、发布、回滚、发布记录、审计。

### 6.4 日志（`internal/logquery`）

- `source: kubernetes`（MVP，直连 Pod 日志 API）与 `source: loki`（P1）双实现，配置切换。
- 查询参数：namespace / pod / container / from / to / keyword / level / limit / direction。
- 支持上下文（context）与导出（export）；Loki 路径支持 TraceId 搜索。

### 6.5 应用发布（`internal/release`，P1）

- 发布对象：版本来源（镜像 tag / Tekton TaskRun 产物）+ 目标（Deployment/StatefulSet 的镜像或配置）。
- 状态机：`pending → approving → running → succeeded / failed`，支持 `paused`、`rolled_back`。
- 审批：按环境（如 prod 需审批人）配置，审批记录入审计。
- 执行后端抽象 `Executor` 接口：`tekton`（默认）/ `kubectl-image`（无 Tekton 时的降级方案）。

### 6.6 AI 诊断（`internal/ai`，P1）

- 入口：Pod 详情、Deployment 详情、日志页、事件页。
- 流程：收集 k8s 上下文（状态/事件/重启次数/最近日志/变更记录）→ 脱敏 → 错误聚合采样 → 规则引擎先判 → 组装 Prompt → 调 LLM → 结构化报告（JSON）→ 落库 → 反馈机制。
- 三档演进：① 规则诊断（不依赖 LLM）→ ② 规则 + LLM 报告 → ③ 历史诊断知识沉淀（检索式，P2）。

### 6.7 资产与远程（`internal/asset`，P2）

- 资产模型：类型（SSH 主机 / Windows RDP / k8s Pod）、凭据（加密存储）、归属、授权关系。
- 会话：WebSSH（xterm.js + WebSocket）、RDP（Guacamole 网关），全程屏幕录像流式落盘。
- 文件管理：SFTP 文件浏览 / 上传 / 下载（仅授权资产）。
- Pod 登录审计：WebSSH 进 Pod 的会话独立记录（谁、何时、哪个 pod、命令轨迹）。

### 6.8 审计（`internal/audit`，横切）

- 中间件自动记录 REST 写操作；终端会话单独录流。
- 存储：平台数据库 + 录像文件目录（PVC），录像文件按会话 id 关联。

---

## 7. 数据模型（核心表，SQLite/PostgreSQL）

```sql
-- 用户 / 角色 / 集群（多集群预留）
users(id, username, email, password_hash, mfa_secret, provider, status, created_at)
roles(id, name)            -- viewer/operator/configadmin/auditor/admin
user_roles(user_id, role_id)
clusters(id, name, kubeconfig_ref, in_cluster, status, created_at)  -- P2 启用

-- 配置中心
apps(id, name, namespace, description, created_at)
environments(id, app_id, name, namespace)
config_items(id, app_id, env_id, config_key, config_value, is_secret, updated_by, updated_at)
config_versions(id, app_id, env_id, version, snapshot, created_by, created_at)
config_releases(id, app_id, env_id, version_id, status, release_note, created_by, created_at)

-- 发布 / 审计 / 诊断 / 资产
releases(id, app_id, env_id, target_kind, target_name, image, status, pipeline_ref, approver, created_at)
audit_logs(id, user_id, action, resource_type, resource_name, namespace, request_body, ip, created_at)
diagnosis_tasks(id, namespace, pod_name, workload_kind, workload_name, time_range, status, input_context, ai_report, feedback, created_by, created_at)
assets(id, name, type, host, port, credential_ref, owner_id, status, created_at)
sessions(id, user_id, asset_id, type, started_at, ended_at, recording_path, command_log)
```

> 完整 DDL 见 `internal/store/migrations/`（001 用户权限、002 集群、003 配置中心、004 发布、005 审计、006 资产会话、007 诊断）。

---

## 8. API 设计约定

统一前缀 `/api/v1`，REST 风格；WebSocket 端点 `/ws/*`。

```text
# 认证
POST /api/v1/auth/login          POST /api/v1/auth/mfa/verify
POST /api/v1/auth/refresh         GET  /api/v1/auth/profile
POST /api/v1/auth/mfa/setup       GET  /api/v1/auth/mfa/status
POST /api/v1/auth/mfa/enrollment  POST /api/v1/auth/mfa/enable|disable
DELETE /api/v1/users/{id}/mfa?confirm=true  # Admin 重置他人 MFA

# 资源
GET  /api/v1/clusters            # 多集群预留（MVP 固定返回当前集群）
GET  /api/v1/namespaces
GET  /api/v1/namespaces/{ns}
GET  /api/v1/nodes
GET  /api/v1/nodes/{name}
GET  /api/v1/resources/yaml?kind=&namespace=&name=
GET  /api/v1/namespaces/{ns}/pods
GET  /api/v1/namespaces/{ns}/pods/{pod}
GET  /api/v1/namespaces/{ns}/pods/{pod}/logs
GET  /api/v1/namespaces/{ns}/pods/{pod}/yaml
DELETE /api/v1/namespaces/{ns}/pods/{pod}
POST /api/v1/namespaces/{ns}/pods/{pod}/restart?confirm=true
POST /api/v1/namespaces/{ns}/deployments/{name}/scale|restart|rollback
GET  /api/v1/namespaces/{ns}/statefulsets[/{name}]
GET  /api/v1/namespaces/{ns}/statefulsets/{name}/yaml
POST /api/v1/namespaces/{ns}/statefulsets/{name}/scale|restart
GET  /api/v1/namespaces/{ns}/daemonsets[/{name}]
GET  /api/v1/namespaces/{ns}/daemonsets/{name}/yaml
POST /api/v1/namespaces/{ns}/daemonsets/{name}/restart
GET  /api/v1/namespaces/{ns}/replicasets
GET  /api/v1/namespaces/{ns}/replicasets/{name}
GET  /api/v1/namespaces/{ns}/jobs
GET  /api/v1/namespaces/{ns}/jobs/{name}
GET  /api/v1/namespaces/{ns}/cronjobs
GET  /api/v1/namespaces/{ns}/cronjobs/{name}
POST /api/v1/namespaces/{ns}/cronjobs/{name}/suspend?confirm=true
POST /api/v1/namespaces/{ns}/cronjobs/{name}/resume?confirm=true
GET  /api/v1/namespaces/{ns}/services
GET  /api/v1/namespaces/{ns}/services/{name}
GET  /api/v1/namespaces/{ns}/ingresses
GET  /api/v1/namespaces/{ns}/ingresses/{name}
GET  /api/v1/namespaces/{ns}/configmaps
GET  /api/v1/namespaces/{ns}/secrets
GET  /api/v1/namespaces/{ns}/secrets/{name}
POST /api/v1/namespaces/{ns}/secrets/{name}/values/{key}?confirm=true
GET  /api/v1/namespaces/{ns}/persistentvolumeclaims
GET  /api/v1/namespaces/{ns}/persistentvolumeclaims/{name}
GET  /api/v1/persistentvolumes
GET  /api/v1/persistentvolumes/{name}
GET  /api/v1/storageclasses
GET  /api/v1/namespaces/{ns}/events

# 配置中心
GET/POST /api/v1/config/apps
GET/PUT  /api/v1/config/apps/{app}/envs/{env}/items
POST     /api/v1/config/apps/{app}/envs/{env}/versions
POST     /api/v1/config/apps/{app}/envs/{env}/release|rollback
GET      /api/v1/config/apps/{app}/envs/{env}/releases

# 日志（source 可切换）
GET  /api/v1/logs?namespace=&pod=&container=&from=&to=&keyword=&level=
GET  /api/v1/logs/contexts
POST /api/v1/logs/export

# 发布（P1）
GET/POST /api/v1/releases
POST     /api/v1/releases/{id}/approve|reject|pause|resume|rollback

# AI（P1）
POST /api/v1/ai/diagnose/pod|deployment|logs
GET  /api/v1/ai/diagnosis/{id}      GET /api/v1/ai/diagnosis/history
POST /api/v1/ai/diagnosis/{id}/feedback

# 资产 / 远程（P2）
GET/POST /api/v1/assets
POST     /api/v1/assets/{id}/authorize
WS   /ws/ssh        WS /ws/pod/exec     WS /ws/rdp
GET  /api/v1/sessions/{id}/recording    # 录像回放

# 审计
GET /api/v1/audit/logs?user=&action=&resource=&from=&to=
```

统一响应：`{ "code": 0, "message": "ok", "data": ... }`；分页参数 `page` / `page_size`；错误码分段（见 `docs/api-conventions.md`）。

---

## 9. 项目目录结构（monorepo）

```text
ops-platform/
├── README.md
├── Makefile
├── go.mod / go.sum
├── cmd/
│   ├── ops-server/main.go        # 后端主入口（可内嵌 web/dist 静态资源）
│   └── ops-migrate/main.go       # 数据库迁移
├── internal/
│   ├── server/                   # 启动、路由、中间件（auth/rbac/audit/recovery/cors）
│   │   └── config/
│   ├── auth/                     # LDAP/OIDC/email、JWT、MFA(TOTP)、角色权限
│   ├── cluster/                  # 多集群抽象：注册表 + client 工厂（MVP 仅 in-cluster）
│   ├── k8s/                      # client 构建、informer、dynamic client、kubeconfig 解析
│   ├── resources/                # namespace/node/pod/service/ingress/pv/pvc/sc/cm/secret/event
│   ├── workload/                 # deployment/sts/ds/pod 的受控写操作与安全边界
│   ├── configcenter/             # app/env/item/version/release/diff/rollback/sync_k8s
│   ├── logquery/                 # source_kube.go / source_loki.go / filter / export
│   ├── release/                  # 发布状态机 + executor(tekton/kubectl-image) + 审批
│   ├── monitor/                  # metrics-server 查询封装（P1）
│   ├── alert/                    # 事件规则引擎 + webhook（P1）
│   ├── ai/                       # 规则诊断 / context_builder / prompt / llm_client / sanitizer
│   ├── asset/                    # 资产模型 / 授权 / ssh+rdp 网关 / 录像（P2）
│   ├── terminal/                 # WebSSH、Pod exec、会话管理（录像接口）
│   ├── audit/                    # 操作审计、会话录像元数据（横切）
│   ├── store/                    # 存储接口 + sqlite/postgres 实现 + 方言迁移
│   ├── model/                    # ORM/表结构
│   └── pkg/                      # logger / errors / pagination / response / websocket / crypto
├── api/openapi/ops-platform.yaml
├── web/                          # Vue 3 + Vite + Element Plus
│   └── src/{api,components,pages,router,stores,utils}
├── deploy/
│   ├── k3s/                      # namespace/rbac/server/web/postgres/ingress
│   │                             #   + optional/sqlite-pvc、fluent-bit/loki/tekton/guacamole（按需）
│   └── helm/ops-platform/        # 可演进
├── configs/ops-server.example.yaml
├── scripts/{build,deploy,backup,restore}.sh
├── docs/                         # architecture / rbac / config-center / audit / deployment / backup
└── test/{integration,e2e}
```

---

## 10. 部署架构（k3s）

### 10.0 组件部署清单（完成平台需部署的全部组件）

按「阶段 + 必选/可选」划分，**P0 闭环只需要 4 类组件**，其余按路线图逐步加装：

| 组件 | 部署形态 | 目标 namespace | 阶段 / 必选 | 作用 |
|---|---|---|---|---|
| **ops-server** | Deployment ×1 | ops-platform | **P0 必选** | 后端主服务：资源/配置/日志/发布/审计/AI 全部 API；可内嵌前端静态资源 |
| **ops-web** | Deployment ×1（或内嵌进 ops-server） | ops-platform | **P0 必选** | 前端控制台（Vue 3 静态资源 + Nginx/静态托管） |
| **PostgreSQL** | StatefulSet ×1 + PVC | ops-platform | **P0 必选** | k3s 正式部署的平台数据：用户/角色/配置版本/发布记录/审计/诊断记录 |
| SQLite PVC | PVC（local-path） | ops-platform | P0 可选 | 本地开发或轻量回退；默认 k3s 部署不安装 |
| **平台 RBAC** | ServiceAccount + ClusterRole | ops-platform | **P0 必选** | 平台访问 k3s API 的最小权限（见 `deploy/k3s/rbac.yaml`） |
| **Ingress（Traefik）** | k3s 自带组件 | kube-system | **P0 必选** | 对外 HTTPS 入口（k3s 默认已装，仅需配置） |
| **Fluent Bit** | DaemonSet | ops-platform | **P1 可选** | 采集 `/var/log/containers/*.log`，带 k8s metadata 推送 Loki |
| **Loki** | Deployment ×1 | ops-platform | **P1 可选** | 日志存储与查询，保留 3~7 天，单实例即可 |
| **metrics-server** | Deployment | kube-system | P1 可选（k3s 默认已装） | 提供 Node/Pod 的 CPU/Memory，支撑容器监控页 |
| **Tekton Pipelines** | Deployment（controller + webhook） | tekton-pipelines | **P1 可选** | 应用发布流水线引擎（CRD 体系，需限制资源） |
| Guacamole | Deployment ×1 | ops-platform | P2 可选 | RDP 网关，支撑 Windows 资产远程（SSH 可自研轻量网关替代） |
| LDAP / 企业微信（OIDC） | 外部服务 | - | P0/P1 可选 | 认证源（`provider` 抽象，本地 email 登录不依赖它） |
| LLM API / Ollama | 外部 / 独立节点 | - | P1 可选 | AI 诊断模型后端（OpenAI-Compatible 协议） |

**组件依赖关系**（决定启动顺序）：

```text
ops-server ──▶ k3s API（必须）  +  PostgreSQL/SQLite（必须）  +  [Loki：仅当 log.source=loki]
Fluent Bit ──▶ Loki（日志链路）
ops-server ──▶ Tekton（仅当启用发布执行后端）
ops-web   ──▶ ops-server（反向代理 /api、/ws）
Guacamole ──▶ ops-server（P2 远程网关，需先部署）
```

**按阶段的最小组件组合**：

```text
P0（MVP 闭环）：k3s + ops-server + ops-web + PostgreSQL StatefulSet/PVC + 平台 RBAC + Ingress
P1（正式版）  ：P0 + Fluent Bit + Loki + metrics-server [+ Tekton]
P2（增强版）  ：P1 + Guacamole
```

> 默认组件清单位于 `deploy/k3s/`；SQLite PVC 等非默认组件位于 `deploy/k3s/optional/`，不会被普通的 `kubectl apply -f deploy/k3s` 自动安装。

### 10.1 部署形态决策：容器化（推荐）

**结论：平台自身容器化、托管在 k3s 内（`ops-platform` namespace），不做主机（systemd）部署。**

这是唯一符合本项目场景（单节点 k3s、规模小）的形态，理由与风险对策如下：

**为什么容器化：**

1. **权限模型天然契合**：平台用 in-cluster ServiceAccount + ClusterRole 访问 k3s API，无需管理 kubeconfig/证书，也没有外网暴露端口问题。
2. **统一生命周期**：Deployment 滚动升级、回滚、自愈（重启）、资源配额，与业务应用同一套标准、同一套 `kubectl apply`。
3. **运维同源**：Fluent Bit 采集日志天然覆盖平台自身日志——平台自己监控自己，不额外加探针。
4. **备份统一**：平台 PostgreSQL 数据使用 PVC 持久化，并通过 `pg_dump` 纳入主机侧备份流程。

**平台「自己托管自己」的风险与对策：**

| 风险 | 对策 |
|---|---|
| 平台挂了没人能修 | Deployment 自愈自动重启；k3s 节点上 `kubectl` 永远可用，可手工 `kubectl -n ops-platform` 兜底 |
| k3s 挂了平台也挂 | 单节点固有风险，**与部署形态无关**——主机部署同样依赖 k3s 活着；靠 k3s 数据备份 + 平台 DB 备份恢复（见第 12 节） |
| 数据持久化 | PostgreSQL 使用独立 local-path PVC；`pg_dump` 备份任务放**主机 cron**并复制到节点外，平台不可用时仍能恢复 |
| 平台吃光业务资源 | requests/limits 严格限制（见 10.4 资源配额参考），与业务同节点隔离由 k3s 调度保证 |
| 启动依赖/循环 | ops-server 是普通 Deployment，k3s 就绪后自动调度，无强依赖顺序 |

**何时才考虑主机部署：**仅当诉求是「集群故障时平台必须仍可访问」（强 HA 场景）才值得。单节点 k3s 下该诉求本身不成立（集群挂了业务也没了，平台活着意义有限），主机部署反而多出 systemd、证书、升级三份维护成本——属于反模式。

---

### 10.2 最简 MVP（够用先上线）

```text
ops-platform namespace
├── ops-server Deployment（1 副本，内嵌前端静态资源）
├── ops-postgres StatefulSet（单副本）+ PVC
├── ServiceAccount + ClusterRole（最小权限）
└── Ingress（Traefik）或 NodePort
```

### 10.3 正式单节点版（P1 目标）

```text
ops-platform namespace
├── ops-server Deployment（含前端或独立 ops-web）
├── ops-web Deployment（可选，若前后端分离）
├── fluent-bit DaemonSet
├── loki Deployment（单实例，保留 3~7 天）
├── PostgreSQL StatefulSet + PVC
└── 可选：tekton-pipelines（独立 namespace，限制资源）
```

### 10.4 资源配额参考

| 组件 | requests | limits |
|---|---|---|
| ops-server | 100m / 256Mi | 500m / 512Mi |
| ops-web | 50m / 128Mi | 200m / 256Mi |
| postgres | 100m / 256Mi | 500m / 512Mi |
| fluent-bit | 50m / 64Mi | 200m / 256Mi |
| loki | 100m / 256Mi | 500m / 768Mi |
| tekton（若启用） | 控制面整体 ≤ 1C / 1Gi | 按需 |

---

## 11. 安全与审计设计

1. **认证**：JWT（短时效 + refresh 轮换）；MFA 支持 TOTP；LDAP 密码不明文落库。
2. **平台 RBAC**：五级角色矩阵（见 6.1），接口层 + 前端路由双层校验。
3. **k8s 权限**：平台自身 ServiceAccount 最小授权；`pods/exec`、`secrets` 写、`nodes` 操作等默认关闭，按需开启并记录审计。
4. **脱敏**：日志查询与 AI 上下文统一经过 `sanitizer`（关键词正则替换）；Secret 内容列表脱敏，仅授权用户可见明文。
5. **审计**：写操作全量记录；WebSSH/RDP 会话流式录像 + 命令日志，防抵赖；审计库只追加。
6. **传输**：对外只暴露 HTTPS（Ingress + TLS）；内部服务不开放 NodePort（除非 MVP 临时）。

---

## 12. 备份与日常运维

- **k3s 数据**：`/var/lib/rancher/k3s/server/db` 定时打包（或 `k3s etcd snapshot`，取决于 datastore 类型）。
- **平台数据**：PostgreSQL 用 `pg_dump`；本地 SQLite 用 `sqlite3 .backup`；录像文件目录一并备份。
- **日志治理**：k3s(containerd) 配置容器日志轮转上限；Loki 保留 3~7 天；采集只选业务 namespace 并做行截断。
- **磁盘规划**：`/`（系统）、`/var/lib/rancher/k3s`（k3s 数据）、`/data/ops-platform`（平台数据）、`/backup`（备份，建议独立盘或远程）。
- **例行任务**：`scripts/backup.sh` + cron；恢复演练每季度一次。

---

## 13. 实施路线图

| 阶段 | 内容 | 周期 |
|---|---|---|
| 0 | 工程骨架：go module、配置加载、SQLite/PostgreSQL 存储与迁移、Makefile、CI、前端脚手架、统一响应/错误码 | 1~2 周 |
| 1（P0） | 认证/权限 + 资源管理 + Pod 日志 + 操作审计（MVP 闭环） | 3~6 周 |
| 2（P1） | 配置中心：模型、发布、回滚、审计 | 4~6 周 |
| 3（P1） | 应用发布：Tekton 集成、审批、回退/暂停（可与阶段 2 并行） | 3~5 周 |
| 4（P1） | 日志平台：Fluent Bit + Loki、查询/上下文/导出 | 3~5 周 |
| 5（P1） | 容器监控 + 事件中心规则告警 + AI 诊断（先规则后 LLM） | 4~8 周 |
| 6（P2） | 资产 + 远程连接（SSH/RDP/录像/文件）+ WebSSH 审计 + 端口转发 + 多集群启用 | 6~10 周 |

> 每阶段结束都是一个可上线的版本；AI、Tekton、Guacamole 均为可选组件，不影响核心闭环。

---

## 14. 开发环境与快速开始

前置：Go ≥ 1.22、Node ≥ 20（或 ≥ 18 LTS）、kubectl、可访问的 k3s/k8s 集群（本地开发可用 k3d/k3s 单机）。

```bash
# 1. 克隆并安装依赖
make deps

# 2. 本地起后端（默认读取 ~/.kube/config，SQLite 落本地文件）
cp configs/ops-server.example.yaml configs/ops-server.yaml
make run-server

# 3. 本地起前端（代理 /api 与 /ws 到后端）
cd web && npm install && npm run dev

# 4. 迁移与初始化
make migrate

# 5. 可选：使用临时 k3d 集群运行资源接口集成测试
# 需要本机已安装并启动 Docker，同时提供 k3d、kubectl、curl。
make test-integration

# 6. 构建与部署到 k3s
make build
make deploy   # kubectl apply -f deploy/k3s
```

服务器侧 P0 统一验收可使用 `scripts/k3s-docker/verify-p0.sh`；默认不执行 PostgreSQL Pod 重建等破坏性步骤，公网入口通过 `PUBLIC_BASE_URL` 显式验证。详细前置条件和开关见 `scripts/k3s-docker/README-DEPLOY.md`。

k3s 清单默认部署单副本 PostgreSQL StatefulSet；数据库凭据由 `deploy/k3s/server-secret.yaml` 注入。存储配置、迁移和备份说明见 `docs/storage.md` 与 `docs/backup.md`。

常用命令见 Makefile：`make lint / test / test-integration / build / image / migrate / backup / restore`。

---

## 15. 工程规范（开发前约定）

1. **分支模型**：`main`（可发布）+ `feature/*` → PR 合入；提交信息遵循 Conventional Commits（`feat|fix|docs|refactor|test|chore`）。
2. **代码分层**：`handler → service → repository/store`，handler 不做业务逻辑；错误统一 `internal/pkg/errors`。
3. **API 变更**：先改 `api/openapi/ops-platform.yaml`，前后端按契约联调。
4. **资源操作安全**：删除类接口二次确认（前端 dialog + 后端 `?confirm=true`）；高危操作（删 namespace、改 Secret）需 Admin 角色。
5. **测试**：store 与 configcenter 必须有单测；资源/发布接口有 integration 测试（依赖 k3s 环境，CI 中用 k3d）。
6. **文档**：每个模块有 `docs/<module>.md`；配置项改动同步 `configs/ops-server.example.yaml`。

---

## 16. 相关文档

- [docs/architecture.md](docs/architecture.md) — 架构详解与演进说明
- [docs/auth.md](docs/auth.md) — 本地认证、JWT 与 TOTP MFA 流程
- [docs/resources.md](docs/resources.md) — P0 资源详情关联、Pod 日志与受控工作负载操作
- [docs/p0-status.md](docs/p0-status.md) — P0 五层覆盖矩阵、真实完成度与不遗漏开发顺序
- [docs/development-checklist.md](docs/development-checklist.md) — 总体阶段、全部功能完成度、未完成项和固定开发顺序
- [docs/rbac.md](docs/rbac.md) — 角色权限矩阵与 k8s 最小授权清单
- [docs/config-center.md](docs/config-center.md) — 配置中心模型与发布流程
- [docs/audit.md](docs/audit.md) — 审计、会话录像与合规设计
- [docs/deployment.md](docs/deployment.md) — k3s 部署与升级
- [docs/backup.md](docs/backup.md) — 备份与恢复
- [docs/storage.md](docs/storage.md) — SQLite/PostgreSQL 驱动、迁移与切换
