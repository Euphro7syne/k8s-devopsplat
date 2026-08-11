package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"ops-platform/internal/config"
	"ops-platform/internal/model"
)

func TestMigrateCreatesBaseTables(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, config.DatabaseConfig{
		Driver:       "sqlite",
		DSN:          filepath.Join(t.TempDir(), "ops.db"),
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}, nil)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var roles int
	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM roles").Scan(&roles); err != nil {
		t.Fatalf("count roles: %v", err)
	}
	if roles != 5 {
		t.Fatalf("expected 5 roles, got %d", roles)
	}

	audit := &model.AuditLog{
		Action:       "POST",
		ResourceType: "http",
		ResourceName: "/api/v1/example",
		IP:           "127.0.0.1",
	}
	if err := store.CreateAuditLog(ctx, audit); err != nil {
		t.Fatalf("create audit log: %v", err)
	}
	if audit.ID == 0 {
		t.Fatalf("expected audit id to be assigned")
	}
}
