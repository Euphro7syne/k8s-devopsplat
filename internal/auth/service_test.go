package auth

import (
	"context"
	"path/filepath"
	"strings"
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

	service.cfg.LocalAdmin.Password = "admin123"
	if err := service.BootstrapAdmin(ctx); err != nil {
		t.Fatalf("sync configured admin password: %v", err)
	}
	if _, err := service.Login(ctx, LoginRequest{
		Email:    "admin@example.com",
		Password: "change-me-admin-password",
	}); err == nil {
		t.Fatalf("expected previous admin password to stop working after config sync")
	}
	login, err = service.Login(ctx, LoginRequest{
		Email:    "admin@example.com",
		Password: "admin123",
	})
	if err != nil {
		t.Fatalf("login with synchronized admin password: %v", err)
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

func TestRequiredMFAEnrollmentAndLogin(t *testing.T) {
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
		MFAEnabled:      true,
		MFAIssuer:       "ops-platform-test",
		MFAChallengeTTL: 5 * time.Minute,
	})
	fixed := time.Unix(1_700_000_000, 0).UTC()
	service.totp.nowFunc = func() time.Time { return fixed }
	service.tokens.nowFunc = func() time.Time { return fixed }

	user, err := service.CreateLocalUser(ctx, CreateUserRequest{
		Username: "operator",
		Email:    "operator@example.com",
		Password: "change-me-operator-password",
		Roles:    []string{"operator"},
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	login, err := service.Login(ctx, LoginRequest{
		Email:    user.Email,
		Password: "change-me-operator-password",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !login.MFARequired || !login.MFASetupRequired || login.MFAToken == "" {
		t.Fatalf("expected required MFA setup challenge, got %#v", login)
	}
	if login.AccessToken != "" || login.RefreshToken != "" {
		t.Fatalf("tokens must not be issued before MFA verification")
	}

	setup, err := service.SetupMFA(ctx, login.MFAToken)
	if err != nil {
		t.Fatalf("setup mfa: %v", err)
	}
	code, err := generateTOTP(setup.Secret, uint64(fixed.Unix()/totpPeriod), totpDigits)
	if err != nil {
		t.Fatalf("generate totp: %v", err)
	}
	verified, err := service.VerifyMFA(ctx, MFAVerifyRequest{MFAToken: setup.MFAToken, Code: code})
	if err != nil {
		t.Fatalf("verify mfa setup: %v", err)
	}
	if verified.AccessToken == "" || verified.RefreshToken == "" || !verified.User.MFAEnabled || !verified.User.MFAVerified {
		t.Fatalf("expected MFA-authenticated token pair, got %#v", verified)
	}
	storedUser, err := db.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("load enrolled user: %v", err)
	}
	if storedUser.MFASecret == setup.Secret || !strings.HasPrefix(storedUser.MFASecret, encryptedMFASecretPrefix) {
		t.Fatalf("expected TOTP secret to be encrypted at rest")
	}

	nextLogin, err := service.Login(ctx, LoginRequest{
		Email:    user.Email,
		Password: "change-me-operator-password",
	})
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	if !nextLogin.MFARequired || nextLogin.MFASetupRequired {
		t.Fatalf("expected verification-only challenge, got %#v", nextLogin)
	}
	invalidCode := "000000"
	if code == invalidCode {
		invalidCode = "000001"
	}
	if _, err := service.VerifyMFA(ctx, MFAVerifyRequest{MFAToken: nextLogin.MFAToken, Code: invalidCode}); err == nil {
		t.Fatalf("expected invalid TOTP to fail")
	}
	verifiedAgain, err := service.VerifyMFA(ctx, MFAVerifyRequest{MFAToken: nextLogin.MFAToken, Code: code})
	if err != nil {
		t.Fatalf("verify login mfa: %v", err)
	}
	if _, err := service.Refresh(ctx, verifiedAgain.RefreshToken); err != nil {
		t.Fatalf("refresh MFA-authenticated session: %v", err)
	}
	if _, err := service.DisableMFA(ctx, user.ID, MFADisableRequest{
		Password: "change-me-operator-password",
		Code:     code,
	}); err == nil {
		t.Fatalf("expected required MFA policy to prevent disabling")
	}
	reset, err := service.ResetUserMFA(ctx, 999, user.ID, true)
	if err != nil {
		t.Fatalf("admin reset mfa: %v", err)
	}
	if reset.MFAEnabled {
		t.Fatalf("expected MFA to be reset")
	}
	afterReset, err := service.Login(ctx, LoginRequest{
		Email:    user.Email,
		Password: "change-me-operator-password",
	})
	if err != nil {
		t.Fatalf("login after reset: %v", err)
	}
	if !afterReset.MFASetupRequired {
		t.Fatalf("expected MFA enrollment after admin reset")
	}
}

func TestDisabledMFASwitchSkipsVerificationAndAllowsBindingCleanup(t *testing.T) {
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
		MFAIssuer:       "ops-platform-test",
		MFAChallengeTTL: 5 * time.Minute,
	})
	fixed := time.Unix(1_700_000_000, 0).UTC()
	service.totp.nowFunc = func() time.Time { return fixed }
	service.tokens.nowFunc = func() time.Time { return fixed }

	user, err := service.CreateLocalUser(ctx, CreateUserRequest{
		Username: "viewer",
		Email:    "viewer@example.com",
		Password: "change-me-viewer-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	login, err := service.Login(ctx, LoginRequest{
		Email:    user.Email,
		Password: "change-me-viewer-password",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if login.MFARequired || login.AccessToken == "" {
		t.Fatalf("expected password-only login while MFA switch is disabled")
	}

	setup, err := service.StartMFAEnrollment(ctx, user.ID)
	if err != nil {
		t.Fatalf("start enrollment: %v", err)
	}
	code, err := generateTOTP(setup.Secret, uint64(fixed.Unix()/totpPeriod), totpDigits)
	if err != nil {
		t.Fatalf("generate totp: %v", err)
	}
	enabled, err := service.EnableMFA(ctx, user.ID, MFAVerifyRequest{MFAToken: setup.MFAToken, Code: code})
	if err != nil {
		t.Fatalf("enable mfa: %v", err)
	}
	if !enabled.User.MFAEnabled || !enabled.User.MFAVerified {
		t.Fatalf("expected enabled MFA principal, got %#v", enabled.User)
	}
	if _, err := service.AuthenticateAccessToken(ctx, login.AccessToken); err != nil {
		t.Fatalf("disabled MFA switch must keep password-only access token valid: %v", err)
	}
	directLogin, err := service.Login(ctx, LoginRequest{
		Email:    user.Email,
		Password: "change-me-viewer-password",
	})
	if err != nil {
		t.Fatalf("login with stored MFA binding while switch is disabled: %v", err)
	}
	if directLogin.MFARequired || directLogin.MFAToken != "" || directLogin.AccessToken == "" {
		t.Fatalf("disabled MFA switch must skip the challenge, got %#v", directLogin)
	}
	if !directLogin.User.MFAEnabled || directLogin.User.MFAVerified {
		t.Fatalf("binding should be reported but not verified while switch is disabled: %#v", directLogin.User)
	}
	if _, err := service.Refresh(ctx, directLogin.RefreshToken); err != nil {
		t.Fatalf("refresh must not require MFA while switch is disabled: %v", err)
	}
	if _, err := service.ResetUserMFA(ctx, user.ID, user.ID, true); err == nil {
		t.Fatalf("expected self MFA reset to be rejected")
	}

	disabled, err := service.DisableMFA(ctx, user.ID, MFADisableRequest{
		Password: "change-me-viewer-password",
		Code:     code,
	})
	if err != nil {
		t.Fatalf("disable mfa: %v", err)
	}
	if disabled.User.MFAEnabled || disabled.User.MFAVerified {
		t.Fatalf("expected MFA-disabled principal, got %#v", disabled.User)
	}
	status, err := service.MFAStatus(ctx, user.ID)
	if err != nil {
		t.Fatalf("get mfa status: %v", err)
	}
	if status.Enabled || status.Required {
		t.Fatalf("unexpected status after disable: %#v", status)
	}

	if err := db.UpdateUserMFASecret(ctx, user.ID, setup.Secret); err != nil {
		t.Fatalf("seed legacy plaintext mfa secret: %v", err)
	}
	legacyLogin, err := service.Login(ctx, LoginRequest{
		Email:    user.Email,
		Password: "change-me-viewer-password",
	})
	if err != nil {
		t.Fatalf("login with legacy mfa secret: %v", err)
	}
	if legacyLogin.MFARequired || legacyLogin.AccessToken == "" {
		t.Fatalf("disabled switch must also skip legacy MFA binding")
	}
	legacyStored, err := db.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("load legacy user while switch is disabled: %v", err)
	}
	if legacyStored.MFASecret != setup.Secret {
		t.Fatalf("disabled switch must not touch the stored MFA secret")
	}

	service.cfg.MFAEnabled = true
	legacyLogin, err = service.Login(ctx, LoginRequest{
		Email:    user.Email,
		Password: "change-me-viewer-password",
	})
	if err != nil {
		t.Fatalf("login after enabling mfa switch: %v", err)
	}
	if !legacyLogin.MFARequired || legacyLogin.MFASetupRequired {
		t.Fatalf("enabled switch must require verification for existing binding")
	}
	if _, err := service.VerifyMFA(ctx, MFAVerifyRequest{MFAToken: legacyLogin.MFAToken, Code: code}); err != nil {
		t.Fatalf("verify legacy mfa secret: %v", err)
	}
	upgraded, err := db.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("load upgraded user: %v", err)
	}
	if !strings.HasPrefix(upgraded.MFASecret, encryptedMFASecretPrefix) {
		t.Fatalf("expected legacy mfa secret to be encrypted after successful verification")
	}
}
