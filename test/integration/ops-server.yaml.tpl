server:
  mode: release
  listen: "__LISTEN__"
  read_timeout: 30s
  write_timeout: 30s
  shutdown_timeout: 10s
  cors:
    allowed_origins:
      - "http://localhost"
    allowed_methods:
      - GET
      - POST
      - PUT
      - PATCH
      - DELETE
      - OPTIONS
    allowed_headers:
      - Authorization
      - Content-Type
      - X-Request-ID

kubernetes:
  mode: kubeconfig
  kubeconfig: "__KUBECONFIG__"
  namespace: ops-platform

database:
  driver: sqlite
  dsn: "file:__DATABASE_PATH__?_foreign_keys=1&_busy_timeout=5000"
  max_open_conns: 1
  max_idle_conns: 1
  auto_migrate: true

log:
  level: info
  format: text

auth:
  jwt_issuer: ops-platform-integration
  jwt_secret: "integration-test-jwt-placeholder"
  access_token_ttl: 15m
  refresh_token_ttl: 1h
  mfa_enabled: false
  mfa_issuer: ops-platform-integration
  mfa_challenge_ttl: 5m
  mfa_secret_key: "integration-test-mfa-placeholder"
  rate_limit:
    enabled: false
  local_admin:
    enabled: true
    username: integration-admin
    email: integration-admin@example.invalid
    password: "integration-test-password"
