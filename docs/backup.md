# 备份

阶段 0 提供 SQLite 备份与恢复脚本：

```bash
make backup
BACKUP_FILE=backup/ops-platform-YYYYMMDDHHMMSS.db make restore
```

生产环境应把 SQLite PVC 与 k3s 数据目录纳入主机侧定时备份。
