# 备份与恢复

## SQLite

```bash
make backup
BACKUP_FILE=backup/ops-platform-YYYYMMDDHHMMSS.db make restore
```

可通过 `DB_PATH` 指定 SQLite 文件路径。

## PostgreSQL

主机需要安装与服务端兼容的 `pg_dump` 和 `pg_restore`。DSN 必须通过环境变量提供，不得写入脚本或提交到仓库：

PostgreSQL Service 仅在集群内可见。从开发机手工备份时，可先建立临时端口转发：

```bash
kubectl -n ops-platform port-forward statefulset/ops-postgres 15432:5432
```

然后在另一个终端执行：

```bash
DB_DRIVER=postgres \
DATABASE_DSN='postgres://ops_platform:<password>@127.0.0.1:15432/ops_platform?sslmode=disable' \
make backup

DB_DRIVER=postgres \
DATABASE_DSN='postgres://ops_platform:<password>@127.0.0.1:15432/ops_platform?sslmode=disable' \
BACKUP_FILE=backup/ops-platform-YYYYMMDDHHMMSS.dump \
make restore
```

恢复会使用 `pg_restore --clean --if-exists` 替换目标数据库中的同名对象，只能对已确认的目标数据库执行。生产环境应定时备份 PostgreSQL，并把备份复制到 PostgreSQL PVC 和 k3s 节点之外。

SQLite 数据不会自动导入 PostgreSQL。已有数据需要在停写窗口内单独执行一次性迁移和数据核对。
