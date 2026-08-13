# 存储

平台通过 `internal/store.Store` 隔离业务层与数据库实现，目前支持 SQLite 和 PostgreSQL。

## 驱动选择

| 驱动 | 配置值 | 适用场景 |
|---|---|---|
| SQLite | `sqlite` / `sqlite3` | 本地开发、轻量单实例运行 |
| PostgreSQL | `postgres` / `postgresql` | k3s 正式部署、写入增长和更强运维能力 |

本地默认仍使用 SQLite。k3s 清单默认部署单副本 PostgreSQL StatefulSet。

配置示例：

```yaml
database:
  driver: postgres
  dsn: "set-by-OPS_DATABASE_DSN"
  max_open_conns: 10
  max_idle_conns: 5
  auto_migrate: true
```

生产环境使用以下环境变量覆盖驱动与 DSN：

```text
OPS_DATABASE_DRIVER=postgres
OPS_DATABASE_DSN=postgres://ops_platform:<password>@ops-postgres:5432/ops_platform?sslmode=disable
```

DSN 包含凭据，必须来自 Kubernetes Secret，不能写入 ConfigMap、日志或代码。

## 迁移

SQLite 和 PostgreSQL 使用独立的方言迁移文件，但共享迁移版本名称。`ops-server` 在 `database.auto_migrate=true` 时自动迁移；也可运行 `make migrate`。

数据库 schema 迁移不等同于数据迁移。把已有 SQLite 数据切到 PostgreSQL 前，需要：

1. 停止平台写入。
2. 备份 SQLite。
3. 初始化 PostgreSQL schema。
4. 转换并导入表数据，同时校正序列。
5. 核对用户、角色、集群和审计记录数量。
6. 修改驱动并启动平台验证。

当前适配不会自动搬迁已有 SQLite 数据，也不会删除 SQLite 文件或 PVC。

## 测试

普通 `make test` 会覆盖 SQLite 和驱动选择逻辑。PostgreSQL 集成测试需要专用测试库：

```bash
OPS_TEST_POSTGRES_DSN='postgres://.../ops_platform_test?sslmode=disable' \
go test ./internal/store/postgres
```

测试库不能指向生产数据库。
