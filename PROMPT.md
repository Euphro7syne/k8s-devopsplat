# PROMPT.md — codex 初始提示词（首次会话直接复制粘贴）

> 用法：把下面「代码块」中的全部内容复制给 codex（或 `codex` 交互式粘贴、或 `codex exec` 传参）。
> 之后每次新会话，只要在根目录运行，codex 会自动读取 `AGENTS.md`，本提示词只需在首个会话使用一次。
> 如需给 codex 更完整的上下文，追加一句：`先通读 README.md 全文，再开始开发。`

---

```
你是一名高级运维开发工程师，负责开发一个 k3s 运维平台（项目代号 ops-platform）。
请先阅读仓库根目录的 AGENTS.md（强制规则）和 README.md（架构权威文档），
有 docs/ 目录时按需阅读对应模块文档，然后再开始开发。

## 项目背景（简版）
- 生产环境是单节点 k3s：约 65 个业务 Pod、4~6 个 Namespace。
- 平台自身也容器化托管在 k3s 的 ops-platform namespace 内（已决策，勿改）。
- 定位是「轻量、模块化、可演进」的模块化单体，禁止引入微服务/重型组件。

## 技术栈（已锁定，不得更换）
- 后端：Go 1.22+、Gin、client-go、informer、gorilla/websocket；存储 SQLite（经 internal/store 接口隔离）。
- 前端：Vue 3、Vite、Element Plus、Pinia、vue-router、xterm.js。
- 日志：MVP 用 K8s API 直查，logquery 模块预留 loki source。

## 本次会话任务：阶段 0 —— 工程骨架（只做这一阶段，不要越界）
1. 初始化 Go module（module 名 ops-platform）与 monorepo 目录结构（严格按 README 第 9 节）。
2. internal/server：启动入口、路由注册、中间件（auth 占位、audit 占位、recovery、cors）。
3. internal/pkg：logger、response（统一 {code,message,data}）、errors（错误码分段）、pagination。
4. internal/store：接口定义 + sqlite 实现 + migrations 目录（001 用户权限 起步）。
5. internal/config：从 configs/ops-server.example.yaml 加载配置（server/kubernetes/database/log/auth 段）。
6. Makefile：deps / run-server / migrate / lint / test / build / image / deploy 目标。
7. 前端：web/ 目录用 Vite 脚手架 Vue 3 + TypeScript + Element Plus，搭好路由、请求封装（/api 代理到后端）、登录页占位。
8. 健康检查：/api/v1/healthz（含 DB 连通性）。
9. CI 骨架（可选）：GitHub Actions 或本地脚本，跑 lint + test + build。

## 交付要求
- 每个模块完成后运行 make lint / make test / make build，必须全部通过（我机器上有 Go 1.22+ 和 Node 20+，直接跑）。
- 提交信息用 Conventional Commits（feat|fix|docs|refactor|test|chore）。
- 不写死任何真实密钥/凭据，一律占位符；新配置项同步进 configs/ops-server.example.yaml。
- 完成后给我一份简短的「已交付清单 + 下一步建议（阶段 1 资源运维 MVP 怎么切）」。
```

---

## 后续会话建议（不需要再用上面整段）

每个新任务用更短的任务式提示即可，例如：

```
你是 ops-platform 的开发工程师。先读 AGENTS.md 和 README.md，再完成：
【阶段 1 / 某模块】实现 XXXX 功能（参考 README 第 6.x 节与 docs/ 对应文档）。
要求：分层 handler→service→store、带单测、make lint && make test && make build 通过、Conventional Commits 提交。
不要越界到其他阶段/模块。
```

## 给 codex 的附加建议（可选配置）

- 在仓库根目录运行 codex，它会自动加载 `AGENTS.md`。
- 若希望 codex 不允许修改某些目录，可在 `AGENTS.md` 或 codex 配置里声明只读路径（如 `docs/`、`*.rtf`）。
- 大型任务拆成多次会话：骨架 → 认证 → 资源管理 → 配置中心 → …（对应 README 第 13 节路线图），一次会话只推进一个阶段，codex 上下文不易失控。
