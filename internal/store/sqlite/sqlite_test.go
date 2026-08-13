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

func TestUserRoleManagement(t *testing.T) {
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

	user := &model.User{
		Username:     "operator",
		Email:        "operator@example.com",
		PasswordHash: "hash",
		Provider:     "local",
		Status:       "active",
	}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := store.ReplaceUserRoles(ctx, user.ID, []string{"viewer", "operator"}); err != nil {
		t.Fatalf("replace user roles: %v", err)
	}
	if err := store.UpdateUserStatus(ctx, user.ID, "disabled"); err != nil {
		t.Fatalf("update status: %v", err)
	}
	if err := store.UpdateUserPasswordHash(ctx, user.ID, "new-hash"); err != nil {
		t.Fatalf("update password hash: %v", err)
	}
	if err := store.UpdateUserMFASecret(ctx, user.ID, "JBSWY3DPEHPK3PXP"); err != nil {
		t.Fatalf("update mfa secret: %v", err)
	}

	users, err := store.ListUsers(ctx)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected one user, got %d", len(users))
	}
	if users[0].Status != "disabled" {
		t.Fatalf("expected disabled user, got %s", users[0].Status)
	}
	if users[0].PasswordHash != "new-hash" {
		t.Fatalf("expected updated password hash")
	}
	if !users[0].MFAEnabled {
		t.Fatalf("expected MFA-enabled user")
	}
	if len(users[0].Roles) != 2 || !containsRole(users[0].Roles, "viewer") || !containsRole(users[0].Roles, "operator") {
		t.Fatalf("unexpected roles: %#v", users[0].Roles)
	}

	roles, err := store.ListRoles(ctx)
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if len(roles) != 5 {
		t.Fatalf("expected 5 roles, got %d", len(roles))
	}
}

func containsRole(roles []string, expected string) bool {
	for _, role := range roles {
		if role == expected {
			return true
		}
	}
	return false
}
