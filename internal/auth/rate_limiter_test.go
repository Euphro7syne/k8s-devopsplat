package auth

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"ops-platform/internal/config"
	apperrors "ops-platform/internal/pkg/errors"
	"ops-platform/internal/store/sqlite"
)

func TestAuthRateLimiterUsesSourceAndIdentityDimensions(t *testing.T) {
	cfg := config.AuthRateLimitConfig{
		Enabled:            true,
		LoginMaxAttempts:   2,
		LoginWindow:        time.Minute,
		LoginBlockDuration: 2 * time.Minute,
		MFAMaxAttempts:     2,
		MFAWindow:          time.Minute,
		MFABlockDuration:   2 * time.Minute,
	}
	fixed := time.Unix(1_700_000_000, 0).UTC()

	bySource := newAuthRateLimiter(cfg)
	bySource.nowFunc = func() time.Time { return fixed }
	if bySource.recordFailure(bySource.loginKeys("192.0.2.10", "first@example.com"), bySource.login) {
		t.Fatalf("first source failure must not block")
	}
	if !bySource.recordFailure(bySource.loginKeys("192.0.2.10", "second@example.com"), bySource.login) {
		t.Fatalf("shared source IP should be blocked after reaching the threshold")
	}
	if !bySource.blocked(bySource.loginKeys("192.0.2.10", "third@example.com")) {
		t.Fatalf("source IP block should apply across accounts")
	}

	byAccount := newAuthRateLimiter(cfg)
	byAccount.nowFunc = func() time.Time { return fixed }
	if byAccount.recordFailure(byAccount.loginKeys("192.0.2.11", "target@example.com"), byAccount.login) {
		t.Fatalf("first account failure must not block")
	}
	if !byAccount.recordFailure(byAccount.loginKeys("192.0.2.12", "TARGET@example.com"), byAccount.login) {
		t.Fatalf("normalized account should be blocked across source IPs")
	}
	if !byAccount.blocked(byAccount.loginKeys("192.0.2.13", "target@example.com")) {
		t.Fatalf("account block should apply across source IPs")
	}
}

func TestAuthRateLimiterResetAndLazyCleanup(t *testing.T) {
	cfg := config.AuthRateLimitConfig{
		Enabled:            true,
		LoginMaxAttempts:   2,
		LoginWindow:        time.Minute,
		LoginBlockDuration: 2 * time.Minute,
		MFAMaxAttempts:     2,
		MFAWindow:          time.Minute,
		MFABlockDuration:   2 * time.Minute,
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	limiter := newAuthRateLimiter(cfg)
	limiter.nowFunc = func() time.Time { return now }
	keys := limiter.loginKeys("192.0.2.20", "viewer@example.com")

	limiter.recordFailure(keys, limiter.login)
	limiter.reset(keys)
	if len(limiter.entries) != 0 {
		t.Fatalf("successful authentication should clear matching counters")
	}

	limiter.recordFailure(keys, limiter.login)
	now = now.Add(time.Minute + time.Second)
	if limiter.blocked(keys) {
		t.Fatalf("failure outside the window must not remain blocked")
	}
	if len(limiter.entries) != 0 {
		t.Fatalf("expired entries should be removed lazily")
	}
}

func TestLoginRateLimitBlocksThenExpires(t *testing.T) {
	ctx := context.Background()
	db := openAuthRateLimitTestStore(t, ctx)
	defer db.Close()

	service := NewService(db, authRateLimitTestConfig())
	now := time.Unix(1_700_000_000, 0).UTC()
	service.limiter.nowFunc = func() time.Time { return now }
	user, err := service.CreateLocalUser(ctx, CreateUserRequest{
		Username: "operator",
		Email:    "operator@example.com",
		Password: "change-me-operator-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	request := LoginRequest{Email: user.Email, Password: "wrong-password"}
	_, err = service.LoginWithSource(ctx, request, "192.0.2.30")
	requireAuthError(t, err, apperrors.CodeUnauthenticated, http.StatusUnauthorized)
	_, err = service.LoginWithSource(ctx, request, "192.0.2.30")
	requireAuthError(t, err, apperrors.CodeRateLimited, http.StatusTooManyRequests)

	request.Password = "change-me-operator-password"
	_, err = service.LoginWithSource(ctx, request, "192.0.2.31")
	requireAuthError(t, err, apperrors.CodeRateLimited, http.StatusTooManyRequests)

	now = now.Add(2*time.Minute + time.Second)
	if _, err := service.LoginWithSource(ctx, request, "192.0.2.31"); err != nil {
		t.Fatalf("login after block expiry: %v", err)
	}
	request.Password = "wrong-password"
	_, err = service.LoginWithSource(ctx, request, "192.0.2.31")
	requireAuthError(t, err, apperrors.CodeUnauthenticated, http.StatusUnauthorized)
}

func TestMFAVerifyRateLimitBlocksChallengeThenExpires(t *testing.T) {
	ctx := context.Background()
	db := openAuthRateLimitTestStore(t, ctx)
	defer db.Close()

	cfg := authRateLimitTestConfig()
	cfg.MFAChallengeTTL = 10 * time.Minute
	cfg.MFAEnabled = true
	service := NewService(db, cfg)
	now := time.Unix(1_700_000_000, 0).UTC()
	service.limiter.nowFunc = func() time.Time { return now }
	service.tokens.nowFunc = func() time.Time { return now }
	service.totp.nowFunc = func() time.Time { return now }

	user, err := service.CreateLocalUser(ctx, CreateUserRequest{
		Username: "viewer",
		Email:    "viewer@example.com",
		Password: "change-me-viewer-password",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	setup, err := service.StartMFAEnrollment(ctx, user.ID)
	if err != nil {
		t.Fatalf("start mfa enrollment: %v", err)
	}
	code, err := generateTOTP(setup.Secret, uint64(now.Unix()/totpPeriod), totpDigits)
	if err != nil {
		t.Fatalf("generate enrollment code: %v", err)
	}
	if _, err := service.EnableMFA(ctx, user.ID, MFAVerifyRequest{MFAToken: setup.MFAToken, Code: code}); err != nil {
		t.Fatalf("enable mfa: %v", err)
	}

	login, err := service.LoginWithSource(ctx, LoginRequest{
		Email:    user.Email,
		Password: "change-me-viewer-password",
	}, "192.0.2.40")
	if err != nil {
		t.Fatalf("login for mfa challenge: %v", err)
	}
	request := MFAVerifyRequest{MFAToken: login.MFAToken, Code: "000000"}
	if request.Code == code {
		request.Code = "000001"
	}
	_, err = service.VerifyMFAWithSource(ctx, request, "192.0.2.40")
	requireAuthError(t, err, apperrors.CodeUnauthenticated, http.StatusUnauthorized)
	_, err = service.VerifyMFAWithSource(ctx, request, "192.0.2.40")
	requireAuthError(t, err, apperrors.CodeRateLimited, http.StatusTooManyRequests)

	now = now.Add(2*time.Minute + time.Second)
	request.Code, err = generateTOTP(setup.Secret, uint64(now.Unix()/totpPeriod), totpDigits)
	if err != nil {
		t.Fatalf("generate post-block code: %v", err)
	}
	if _, err := service.VerifyMFAWithSource(ctx, request, "192.0.2.41"); err != nil {
		t.Fatalf("verify mfa after block expiry: %v", err)
	}
}

func authRateLimitTestConfig() config.AuthConfig {
	return config.AuthConfig{
		JWTIssuer:       "ops-platform-test",
		JWTSecret:       "test-secret",
		AccessTokenTTL:  time.Minute,
		RefreshTokenTTL: time.Hour,
		MFAIssuer:       "ops-platform-test",
		MFAChallengeTTL: 5 * time.Minute,
		RateLimit: config.AuthRateLimitConfig{
			Enabled:            true,
			LoginMaxAttempts:   2,
			LoginWindow:        time.Minute,
			LoginBlockDuration: 2 * time.Minute,
			MFAMaxAttempts:     2,
			MFAWindow:          time.Minute,
			MFABlockDuration:   2 * time.Minute,
		},
	}
}

func openAuthRateLimitTestStore(t *testing.T, ctx context.Context) *sqlite.Store {
	t.Helper()
	db, err := sqlite.Open(ctx, config.DatabaseConfig{
		Driver:       "sqlite",
		DSN:          filepath.Join(t.TempDir(), "ops.db"),
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}, nil)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Migrate(ctx); err != nil {
		db.Close()
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func requireAuthError(t *testing.T, err error, code apperrors.Code, status int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %d", code)
	}
	appErr := apperrors.From(err)
	if appErr.Code != code || appErr.HTTPStatus != status {
		t.Fatalf("expected code/status %d/%d, got %d/%d: %v", code, status, appErr.Code, appErr.HTTPStatus, err)
	}
}
