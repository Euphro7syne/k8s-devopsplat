# 认证与 MFA

阶段 1 当前实现本地 email + 密码登录、JWT access/refresh 会话、平台 RBAC，以及 RFC 6238 TOTP 多因素认证。LDAP/OIDC 仍是后续认证源扩展，不在本阶段实现范围内。

## MFA 策略

`auth.mfa_enabled` 是 TOTP 登录认证的全局硬开关：

- `false`：登录只校验 email 和密码，不返回 MFA challenge，也不要求任何已绑定账号输入动态码。已有 TOTP 绑定数据保留，便于后续重新开启开关时继续使用。
- `true`：所有账号必须使用 TOTP。尚未绑定的账号在密码校验成功后进入首次绑定流程，绑定验证成功前不会签发 access/refresh token；已绑定账号必须输入动态码。

TOTP 参数固定为 SHA-1、30 秒周期、6 位数字，校验允许前后各一个时间窗口以容忍小幅时钟偏差。服务端使用加密安全随机源生成 160-bit Base32 密钥，并返回标准 `otpauth://` URI，前端渲染二维码供兼容身份验证器扫描。绑定完成后，TOTP 密钥使用 AES-GCM 加密再写入平台数据库（SQLite/PostgreSQL）。

## 登录链路

```text
email + password
  ├─ 无需 MFA ───────────────────────▶ access + refresh token
  └─ 需要 MFA ─▶ 5 分钟 MFA challenge
                   ├─ 已绑定 ─▶ 校验 6 位动态码 ─▶ access + refresh token
                   └─ 未绑定 ─▶ 生成二维码/密钥
                                  └─ 首次动态码校验并保存绑定
                                     └─ access + refresh token
```

JWT 中记录当前会话是否完成 MFA。平台从关闭切换为开启后，旧的、未完成 MFA 的 access/refresh token 会被拒绝，必须重新登录并完成验证；从开启切换为关闭后，后续登录和 refresh 不再要求动态码。

## 失败速率限制

`POST /api/v1/auth/login` 和 `POST /api/v1/auth/mfa/verify` 默认启用进程内失败速率限制：5 分钟窗口内累计失败 5 次后封禁 15 分钟，并返回 HTTP `429`、错误码 `20003`。登录分别按来源 IP 与规范化账号维度计数，MFA Verify 分别按来源 IP 与 challenge token 维度计数；任一维度达到阈值都会阻止后续尝试。

账号、来源 IP 和 MFA challenge token 在限流器中只保留 SHA-256 摘要，不保存原始值。认证成功后清除本次来源和身份对应的失败计数，窗口或封禁过期的记录由服务惰性回收。

该实现面向当前单节点、单副本部署，不依赖 Redis 或数据库表。服务重启后计数会清零；如果未来把 `ops-server` 扩展为多副本，需要单独设计共享限流状态，不能假定当前内存计数能跨 Pod 同步。

## API

| 接口 | 认证要求 | 用途 |
|---|---|---|
| `POST /api/v1/auth/login` | 公开 | 校验密码；按策略返回 JWT 或 MFA challenge |
| `POST /api/v1/auth/mfa/setup` | MFA challenge | 首次强制绑定时生成 TOTP 密钥和 URI |
| `POST /api/v1/auth/mfa/verify` | MFA challenge | 完成首次绑定或登录二次验证并签发 JWT |
| `GET /api/v1/auth/mfa/status` | access token | 查询个人绑定状态和 TOTP 登录开关 |
| `POST /api/v1/auth/mfa/enrollment` | access token | 创建 TOTP 绑定信息；登录开关关闭时不会影响登录链路 |
| `POST /api/v1/auth/mfa/enable` | access token + TOTP | 验证并保存个人 TOTP 绑定 |
| `POST /api/v1/auth/mfa/disable` | access token + 密码 + TOTP | 登录开关关闭时移除个人 TOTP 绑定 |
| `DELETE /api/v1/users/{id}/mfa?confirm=true` | admin | 重置其他用户的 MFA；下次登录重新绑定 |

MFA 启用、关闭、登录验证和管理员重置均进入写请求审计。审计 sanitizer 会对 JSON 中的 `password`、`mfa_token`、`secret` 和动态 `code` 等字段打码。

## 配置

| 配置项 | 默认值 | 说明 |
|---|---|---|
| `auth.mfa_enabled` | `false` | TOTP 登录硬开关：`true` 必须验证动态码，`false` 只验证账号密码 |
| `auth.mfa_issuer` | `ops-platform` | 身份验证器中展示的签发方名称 |
| `auth.mfa_challenge_ttl` | `5m` | MFA 登录/绑定挑战的有效期 |
| `auth.mfa_secret_key` | `change-me-mfa-secret-key` | TOTP 密钥的 AES-GCM 加密主密钥，生产必须替换并通过 Secret 管理 |
| `auth.rate_limit.enabled` | `true` | 是否启用登录与 MFA Verify 失败速率限制 |
| `auth.rate_limit.login_max_attempts` | `5` | 登录失败达到封禁的次数 |
| `auth.rate_limit.login_window` | `5m` | 登录失败统计窗口 |
| `auth.rate_limit.login_block_duration` | `15m` | 登录封禁时长 |
| `auth.rate_limit.mfa_max_attempts` | `5` | MFA Verify 失败达到封禁的次数 |
| `auth.rate_limit.mfa_window` | `5m` | MFA Verify 失败统计窗口 |
| `auth.rate_limit.mfa_block_duration` | `15m` | MFA Verify 封禁时长 |

默认本地管理员为 `admin@example.com` / `admin123`。可通过 `auth.local_admin.password` 或 `OPS_AUTH_LOCAL_ADMIN_PASSWORD` 修改密码；服务启动时会把该值同步为 SQLite/PostgreSQL 中已有本地管理员的 bcrypt 哈希，修改后重启 `ops-server` 即可生效。生产环境必须覆盖默认管理员密码，并通过 `OPS_AUTH_JWT_SECRET`、`OPS_AUTH_MFA_SECRET_KEY` 管理认证密钥；PostgreSQL 部署还会通过 `OPS_DATABASE_DSN` 注入数据库凭据。k3s 清单从 `ops-server-secrets` Secret 注入这些变量。节点时间应通过 NTP 保持同步，否则 TOTP 校验会受到影响。`mfa_secret_key` 不能随意轮换；轮换前需要提供旧密钥解密/重加密流程。

## 恢复约束

- TOTP 登录开关关闭时，用户登录不再需要动态码；已有绑定可保留，也可使用当前密码和动态码移除。
- 用户丢失身份验证器时，由其他 Admin 在用户管理页重置 MFA；删除类操作要求前端确认和后端 `confirm=true` 双重确认。
- Admin 不能通过用户管理接口重置自己的 MFA，避免在强制策略下立即锁死当前账号。至少应保留两个可用 Admin 账号作为生产恢复措施。
