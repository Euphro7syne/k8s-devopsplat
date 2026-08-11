# AGENTS.md — 项目规则（AI 开发代理必读）

本文件是 `ops-platform`（k3s 运维平台）的长期开发规则。任何 AI 编码代理（codex / 其他）在修改本仓库代码前必须先读本文件与 `README.md`。

## 1. 项目定位

单节点 k3s 运维平台：资源管理、配置中心、日志查询、应用发布、审计、AI 诊断。
规模假设：**1 节点 / ~65 业务 Pod / 4~6 个 Namespace**。一切设计以「轻量、模块化、可演进」为准。

## 2. 技术栈（已锁定，不得擅自更换）

| 层 | 选型 |
|---|---|
| 后端 | Go（≥1.22）+ Gin + client-go + informer + gorilla/websocket |
| 前端 | Vue 3 + Vite + Element Plus + Pinia + vue-router + xterm.js |
| 存储 | SQLite（MVP，`internal/store` 接口隔离，后续可切 PostgreSQL） |
| 日志 | 默认 K8s API 直查；P1 起 Fluent Bit → Loki（`logquery` 支持 source 切换） |
| 监控 | metrics-server 数据封装；**不引入 Prometheus 全家桶** |
| 流水线 | Tekton（P1，通过 `release` 的 Executor 接口抽象，不强耦合） |
| AI | 内部模块，先规则诊断后 LLM（OpenAI-Compatible API），必做脱敏 |

**禁止引入**：微服务拆分、Kafka/RabbitMQ、ClickHouse/OpenSearch/ELK、服务网格、多副本 HA 控制面、前端 SSR/Node 中间层。

## 3. 架构约束

1. **模块化单体**：所有后端逻辑在 `internal/<module>/`，禁止跨模块直接调内部实现，通过 `internal/server` 的路由与 `internal/store` 接口收敛。
2. **分层**：`handler → service → store`，handler 不做业务逻辑；错误统一走 `internal/pkg/errors`；响应统一 `{code, message, data}`。
3. **多集群**：后端保留 `internal/cluster`（Cluster 模型 + client 工厂），**MVP 只实现 in-cluster 单集群**，UI 预留切换入口。
4. **审计横切**：所有写操作必须经过审计中间件落 `audit_logs`；WebSSH/RDP 会话必须录像。
5. **配置中心**：模型固定为 App → Environment → ConfigItem → ConfigVersion(快照) → ConfigRelease；发布 = 渲染 ConfigMap/Secret → Apply → 打 `ops.platform/config-version` annotation 触发滚动更新；禁止平台随意直改 ConfigMap。
6. **安全红线**：
   - 平台 ServiceAccount 使用最小化 ClusterRole；`pods/exec`、`secrets` 写权限默认关闭。
   - 日志/AI 上下文必须过 `sanitizer`（password/token/secret/authorization/api_key/private_key 一律打码）。
   - Secret 平台内 AES-GCM 加密存储、列表脱敏展示。
   - 删除类接口二次确认；高危操作（删 namespace、改 Secret）仅 Admin 角色。

## 4. 工程规范

- **提交**：Conventional Commits（`feat|fix|docs|refactor|test|chore`），中文或英文描述均可。
- **测试**：`store` 与 `configcenter` 必须有单测；资源/发布接口有 integration 测试（CI 用 k3d）；新功能必须带测试。
- **API 契约**：改 API 先更新 `api/openapi/ops-platform.yaml`；前后端按契约联调。
- **文档同步**：新增/修改配置项同步 `configs/ops-server.example.yaml`；模块级改动同步 `docs/<module>.md`。
- **目录**：按 README 第 9 节 monorepo 结构；前端代码只在 `web/`，部署清单只在 `deploy/k3s/`。

## 5. 对 AI 代理的强制要求

1. 动手前先读 `README.md`（权威文档）与本文件；涉及模块先读对应 `docs/`。
2. 一次只做一个阶段/一个任务，不要顺手重构无关代码。
3. 生成代码必须可编译可运行（`make build` / `make test` 通过才算完成）。
4. 不确定的架构取舍先询问，不要自行扩大范围（如擅自加数据库、加组件、改前端框架）。
5. 不写入任何真实密钥/凭据到代码或配置示例中，一律用占位符。
