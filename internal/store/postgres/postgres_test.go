package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"ops-platform/internal/config"
	"ops-platform/internal/model"
)

func TestPostgresStoreIntegration(t *testing.T) {
	dsn := os.Getenv("OPS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPS_TEST_POSTGRES_DSN is not set")
	}

	ctx := context.Background()
	db, err := Open(ctx, config.DatabaseConfig{
		Driver:       "postgres",
		DSN:          dsn,
		MaxOpenConns: 2,
		MaxIdleConns: 1,
	}, nil)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("migrate postgres: %v", err)
	}

	roles, err := db.ListRoles(ctx)
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if len(roles) != 5 {
		t.Fatalf("expected 5 roles, got %d", len(roles))
	}

	unique := time.Now().UnixNano()
	user := &model.User{
		Username:     fmt.Sprintf("postgres-test-%d", unique),
		Email:        fmt.Sprintf("postgres-test-%d@example.com", unique),
		PasswordHash: "hash",
		Provider:     "local",
		Status:       "active",
	}
	if err := db.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	defer func() {
		_, _ = db.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", user.ID)
	}()

	if err := db.ReplaceUserRoles(ctx, user.ID, []string{"viewer", "operator"}); err != nil {
		t.Fatalf("replace roles: %v", err)
	}
	if err := db.UpdateUserMFASecret(ctx, user.ID, "enc:v1:test"); err != nil {
		t.Fatalf("update mfa secret: %v", err)
	}
	if err := db.UpdateUserPasswordHash(ctx, user.ID, "new-hash"); err != nil {
		t.Fatalf("update password hash: %v", err)
	}

	loaded, err := db.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if !loaded.MFAEnabled || len(loaded.Roles) != 2 || loaded.PasswordHash != "new-hash" {
		t.Fatalf("unexpected loaded user: %#v", loaded)
	}

	audit := &model.AuditLog{
		UserID:       &user.ID,
		Action:       "POST",
		ResourceType: "http",
		ResourceName: "/api/v1/postgres-test",
		IP:           "127.0.0.1",
	}
	if err := db.CreateAuditLog(ctx, audit); err != nil {
		t.Fatalf("create audit log: %v", err)
	}
	defer func() {
		_, _ = db.db.ExecContext(ctx, "DELETE FROM audit_logs WHERE id = $1", audit.ID)
	}()
	if audit.ID == 0 {
		t.Fatalf("expected audit id")
	}

	clusters, err := db.ListClusters(ctx)
	if err != nil {
		t.Fatalf("list clusters: %v", err)
	}
	if len(clusters) == 0 || !clusters[0].InCluster {
		t.Fatalf("expected in-cluster seed, got %#v", clusters)
	}
}
