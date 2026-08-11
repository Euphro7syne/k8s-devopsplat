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
