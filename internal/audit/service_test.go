package audit

import (
	"context"
	"path/filepath"
	"testing"

	"ops-platform/internal/config"
	"ops-platform/internal/model"
	"ops-platform/internal/store"
	"ops-platform/internal/store/sqlite"
)

func TestListLogs(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(ctx, config.DatabaseConfig{
		Driver:       "sqlite",
		DSN:          filepath.Join(t.TempDir(), "ops.db"),
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}, nil)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.CreateAuditLog(ctx, &model.AuditLog{Action: "DELETE", ResourceType: "http", ResourceName: "/api/v1/pods", Namespace: "default"}); err != nil {
		t.Fatalf("create audit log: %v", err)
	}

	service := NewService(db)
	result, err := service.ListLogs(ctx, store.AuditLogQuery{Action: "DELETE"})
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected one audit log, got %d", len(result.Items))
	}
}
