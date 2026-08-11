package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
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
	if cfg.Database.Driver != "sqlite" {
		t.Fatalf("unexpected database driver: %s", cfg.Database.Driver)
	}
}
