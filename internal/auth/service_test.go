package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"ops-platform/internal/config"
	"ops-platform/internal/store/sqlite"
)

func TestBootstrapLoginAndRefresh(t *testing.T) {
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

	service := NewService(db, config.AuthConfig{
		JWTIssuer:       "ops-platform-test",
		JWTSecret:       "test-secret",
		AccessTokenTTL:  time.Minute,
		RefreshTokenTTL: time.Hour,
		LocalAdmin: config.LocalAdminConfig{
			Enabled:  true,
			Username: "admin",
			Email:    "admin@example.com",
			Password: "change-me-admin-password",
		},
	})
	if err := service.BootstrapAdmin(ctx); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}

	login, err := service.Login(ctx, LoginRequest{
		Email:    "admin@example.com",
		Password: "change-me-admin-password",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if login.AccessToken == "" || login.RefreshToken == "" {
		t.Fatalf("expected token pair")
	}
	if !HasAnyRole(login.User, "operator") {
		t.Fatalf("admin should satisfy role checks")
	}

	refreshed, err := service.Refresh(ctx, login.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.AccessToken == "" {
		t.Fatalf("expected refreshed access token")
	}
}

func TestUserManagement(t *testing.T) {
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

	service := NewService(db, config.AuthConfig{
		JWTIssuer:       "ops-platform-test",
		JWTSecret:       "test-secret",
		AccessTokenTTL:  time.Minute,
		RefreshTokenTTL: time.Hour,
	})

	user, err := service.CreateLocalUser(ctx, CreateUserRequest{
		Username: "operator",
		Email:    "operator@example.com",
		Password: "change-me-operator-password",
		Roles:    []string{"operator", "viewer"},
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.ID == 0 || len(user.Roles) != 2 {
		t.Fatalf("unexpected created user: %#v", user)
	}

	login, err := service.Login(ctx, LoginRequest{
		Email:    "operator@example.com",
		Password: "change-me-operator-password",
	})
	if err != nil {
		t.Fatalf("login created user: %v", err)
	}
	if !HasAnyRole(login.User, "operator") {
		t.Fatalf("expected operator role")
	}

	if _, err := service.ReplaceUserRoles(ctx, user.ID, user.ID, UpdateUserRolesRequest{Roles: []string{"viewer"}}); err == nil {
		t.Fatalf("expected self role update to be rejected")
	}
	if _, err := service.ReplaceUserRoles(ctx, 999, user.ID, UpdateUserRolesRequest{Roles: []string{"viewer"}}); err != nil {
		t.Fatalf("replace user roles: %v", err)
	}
	authenticated, err := service.AuthenticateAccessToken(ctx, login.AccessToken)
	if err != nil {
		t.Fatalf("authenticate access token: %v", err)
	}
	if HasAnyRole(authenticated, "operator") {
		t.Fatalf("expected current roles to be loaded from store")
	}
	updated, err := service.UpdateUserStatus(ctx, 999, user.ID, UpdateUserStatusRequest{Status: "disabled"})
	if err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if updated.Status != "disabled" {
		t.Fatalf("expected disabled status, got %s", updated.Status)
	}
	if _, err := service.AuthenticateAccessToken(ctx, login.AccessToken); err == nil {
		t.Fatalf("expected disabled user access token to be rejected")
	}
	if _, err := service.Login(ctx, LoginRequest{
		Email:    "operator@example.com",
		Password: "change-me-operator-password",
	}); err == nil {
		t.Fatalf("expected disabled user login to fail")
	}
}
