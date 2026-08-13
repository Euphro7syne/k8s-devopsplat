package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("OPS_DATABASE_DRIVER", "postgres")
	t.Setenv("OPS_DATABASE_DSN", "postgres://ops_platform:placeholder@localhost:5432/ops_platform?sslmode=disable")
	t.Setenv("OPS_AUTH_JWT_SECRET", "env-jwt-secret")
	t.Setenv("OPS_AUTH_MFA_SECRET_KEY", "env-mfa-secret-key")
	t.Setenv("OPS_AUTH_LOCAL_ADMIN_PASSWORD", "env-admin-password")
	path := filepath.Join(t.TempDir(), "ops-server.yaml")
	raw := []byte(`
server:
  listen: "127.0.0.1:18080"
  read_timeout: 3s
database:
  driver: sqlite
  dsn: "file:test.db?_foreign_keys=1"
log:
  level: debug
auth:
  jwt_secret: "placeholder"
  rate_limit:
    login_max_attempts: 7
`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Server.Listen != "127.0.0.1:18080" {
		t.Fatalf("unexpected listen: %s", cfg.Server.Listen)
	}
	if cfg.Server.ReadTimeout != 3*time.Second {
		t.Fatalf("unexpected read timeout: %s", cfg.Server.ReadTimeout)
	}
	if cfg.Database.Driver != "postgres" {
		t.Fatalf("unexpected database driver: %s", cfg.Database.Driver)
	}
	if cfg.Database.DSN != "postgres://ops_platform:placeholder@localhost:5432/ops_platform?sslmode=disable" {
		t.Fatalf("unexpected database dsn: %s", cfg.Database.DSN)
	}
	if cfg.Auth.JWTSecret != "env-jwt-secret" || cfg.Auth.MFASecretKey != "env-mfa-secret-key" {
		t.Fatalf("expected auth secret environment overrides")
	}
	if cfg.Auth.LocalAdmin.Password != "env-admin-password" {
		t.Fatalf("expected local admin password environment override")
	}
	if !cfg.Auth.RateLimit.Enabled || cfg.Auth.RateLimit.LoginMaxAttempts != 7 || cfg.Auth.RateLimit.MFAMaxAttempts != 5 {
		t.Fatalf("expected parsed auth rate limiting with remaining defaults, got %#v", cfg.Auth.RateLimit)
	}
}

func TestValidateDatabaseDrivers(t *testing.T) {
	for _, driver := range []string{"sqlite", "sqlite3", "postgres", "postgresql"} {
		cfg := Default()
		cfg.Database.Driver = driver
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected driver %q to be valid: %v", driver, err)
		}
	}

	cfg := Default()
	cfg.Database.Driver = "mysql"
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected mysql driver to be rejected")
	}
}

func TestValidateAuthRateLimit(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*AuthRateLimitConfig)
	}{
		{name: "login max attempts", mutate: func(cfg *AuthRateLimitConfig) { cfg.LoginMaxAttempts = 0 }},
		{name: "login window", mutate: func(cfg *AuthRateLimitConfig) { cfg.LoginWindow = 0 }},
		{name: "login block duration", mutate: func(cfg *AuthRateLimitConfig) { cfg.LoginBlockDuration = 0 }},
		{name: "mfa max attempts", mutate: func(cfg *AuthRateLimitConfig) { cfg.MFAMaxAttempts = 0 }},
		{name: "mfa window", mutate: func(cfg *AuthRateLimitConfig) { cfg.MFAWindow = 0 }},
		{name: "mfa block duration", mutate: func(cfg *AuthRateLimitConfig) { cfg.MFABlockDuration = 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.mutate(&cfg.Auth.RateLimit)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected invalid rate limit config")
			}
		})
	}

	cfg := Default()
	cfg.Auth.RateLimit.Enabled = false
	cfg.Auth.RateLimit.LoginMaxAttempts = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("disabled rate limiting should ignore policy values: %v", err)
	}
}

func TestDefaultLocalAdminPassword(t *testing.T) {
	cfg := Default()
	if cfg.Auth.LocalAdmin.Password != "admin123" {
		t.Fatalf("unexpected default local admin password: %q", cfg.Auth.LocalAdmin.Password)
	}
}
