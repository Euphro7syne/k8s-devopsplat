# 认证与 MFA

阶段 1 当前实现本地 email + 密码登录、JWT access/refresh 会话、平台 RBAC，以及 RFC 6238 TOTP 多因素认证。LDAP/OIDC 仍是后续认证源扩展，不在本阶段实现范围内。

## MFA 策略

`auth.mfa_enabled` 控制平台是否强制所有用户使用 MFA：

- `false`：用户可在“安全设置”中自愿启用或关闭 MFA；已经启用 MFA 的账号登录时仍必须验证动态码。
- `true`：所有账号必须启用 MFA。尚未绑定的账号在密码校验成功后进入首次绑定流程，绑定验证成功前不会签发 access/refresh token；已绑定账号不能自行关闭 MFA。

TOTP 参数固定为 SHA-1、30 秒周期、6 位数字，校验允许前后各一个时间窗口以容忍小幅时钟偏差。服务端使用加密安全随机源生成 160-bit Base32 密钥，并返回标准 `otpauth://` URI，前端渲染二维码供兼容身份验证器扫描。绑定完成后，TOTP 密钥使用 AES-GCM 加密再写入 SQLite。

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

JWT 中记录当前会话是否完成 MFA。用户启用 MFA 或平台切换为强制 MFA 后，旧的、未完成 MFA 的 access/refresh token 会被拒绝，必须重新登录并完成验证。

## API

| 接口 | 认证要求 | 用途 |
|---|---|---|
| `POST /api/v1/auth/login` | 公开 | 校验密码；按策略返回 JWT 或 MFA challenge |
| `POST /api/v1/auth/mfa/setup` | MFA challenge | 首次强制绑定时生成 TOTP 密钥和 URI |
| `POST /api/v1/auth/mfa/verify` | MFA challenge | 完成首次绑定或登录二次验证并签发 JWT |
| `GET /api/v1/auth/mfa/status` | access token | 查询个人 MFA 状态和平台强制策略 |
| `POST /api/v1/auth/mfa/enrollment` | access token | 为当前用户创建可选 MFA 绑定信息 |
| `POST /api/v1/auth/mfa/enable` | access token + TOTP | 验证并启用个人 MFA，返回替换后的 JWT |
| `POST /api/v1/auth/mfa/disable` | access token + 密码 + TOTP | 在非强制策略下关闭个人 MFA |
| `DELETE /api/v1/users/{id}/mfa?confirm=true` | admin | 重置其他用户的 MFA；下次登录重新绑定 |

MFA 启用、关闭、登录验证和管理员重置均进入写请求审计。审计 sanitizer 会对 JSON 中的 `password`、`mfa_token`、`secret` 和动态 `code` 等字段打码。

## 配置

| 配置项 | 默认值 | 说明 |
|---|---|---|
| `auth.mfa_enabled` | `false` | 是否强制所有用户绑定并验证 MFA |
| `auth.mfa_issuer` | `ops-platform` | 身份验证器中展示的签发方名称 |
| `auth.mfa_challenge_ttl` | `5m` | MFA 登录/绑定挑战的有效期 |
| `auth.mfa_secret_key` | `change-me-mfa-secret-key` | TOTP 密钥的 AES-GCM 加密主密钥，生产必须替换并通过 Secret 管理 |

生产环境必须通过 `OPS_AUTH_JWT_SECRET`、`OPS_AUTH_MFA_SECRET_KEY` 和 `OPS_AUTH_LOCAL_ADMIN_PASSWORD` 环境变量覆盖示例值；k3s 清单从 `ops-server-secrets` Secret 注入这些变量。节点时间应通过 NTP 保持同步，否则 TOTP 校验会受到影响。`mfa_secret_key` 不能随意轮换；轮换前需要提供旧密钥解密/重加密流程。

## 恢复约束

- 用户仍有有效会话且平台未强制 MFA 时，可使用当前密码和动态码关闭 MFA。
- 用户丢失身份验证器时，由其他 Admin 在用户管理页重置 MFA；删除类操作要求前端确认和后端 `confirm=true` 双重确认。
- Admin 不能通过用户管理接口重置自己的 MFA，避免在强制策略下立即锁死当前账号。至少应保留两个可用 Admin 账号作为生产恢复措施。
