package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"ops-platform/internal/config"
	"ops-platform/internal/model"
	apperrors "ops-platform/internal/pkg/errors"
	"ops-platform/internal/store"
)

const (
	mfaPurposeVerify = "verify"
	mfaPurposeEnroll = "enroll"
)

type Service struct {
	store   store.AuthStore
	cfg     config.AuthConfig
	tokens  *TokenManager
	totp    *TOTP
	secrets *MFASecretCipher
	limiter *authRateLimiter
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	TokenPair
	User             Principal `json:"user"`
	MFARequired      bool      `json:"mfa_required"`
	MFASetupRequired bool      `json:"mfa_setup_required"`
	MFAToken         string    `json:"mfa_token,omitempty"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type MFASetupRequest struct {
	MFAToken string `json:"mfa_token" binding:"required"`
}

type MFAVerifyRequest struct {
	MFAToken string `json:"mfa_token" binding:"required"`
	Code     string `json:"code" binding:"required,len=6,numeric"`
}

type MFADisableRequest struct {
	Password string `json:"password" binding:"required"`
	Code     string `json:"code" binding:"required,len=6,numeric"`
}

type MFASetupResponse struct {
	Secret          string `json:"secret"`
	ProvisioningURI string `json:"provisioning_uri"`
	MFAToken        string `json:"mfa_token"`
	ExpiresIn       int64  `json:"expires_in"`
}

type MFAStatus struct {
	Enabled  bool `json:"enabled"`
	Required bool `json:"required"`
}

type CreateUserRequest struct {
	Username string   `json:"username" binding:"required"`
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=8"`
	Roles    []string `json:"roles"`
}

type UpdateUserStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type UpdateUserRolesRequest struct {
	Roles []string `json:"roles" binding:"required"`
}

func NewService(store store.AuthStore, cfg config.AuthConfig) *Service {
	if strings.TrimSpace(cfg.MFAIssuer) == "" {
		cfg.MFAIssuer = strings.TrimSpace(cfg.JWTIssuer)
		if cfg.MFAIssuer == "" {
			cfg.MFAIssuer = "ops-platform"
		}
	}
	if cfg.MFAChallengeTTL <= 0 {
		cfg.MFAChallengeTTL = 5 * time.Minute
	}
	if cfg.MFASecretKey == "" {
		cfg.MFASecretKey = cfg.JWTSecret
		if cfg.MFASecretKey == "" {
			cfg.MFASecretKey = "change-me-mfa-secret-key"
		}
	}
	cfg.RateLimit = cfg.RateLimit.WithDefaults()
	return &Service{
		store:   store,
		cfg:     cfg,
		tokens:  NewTokenManager(cfg),
		totp:    NewTOTP(cfg.MFAIssuer),
		secrets: NewMFASecretCipher(cfg.MFASecretKey),
		limiter: newAuthRateLimiter(cfg.RateLimit),
	}
}

func (s *Service) BootstrapAdmin(ctx context.Context) error {
	if s.store == nil || !s.cfg.LocalAdmin.Enabled {
		return nil
	}
	email := strings.TrimSpace(strings.ToLower(s.cfg.LocalAdmin.Email))
	if email == "" || s.cfg.LocalAdmin.Password == "" {
		return fmt.Errorf("local admin email and password are required")
	}
	if user, err := s.store.GetUserByEmail(ctx, email); err == nil {
		if user.Provider != "local" {
			return fmt.Errorf("local admin email %q belongs to provider %q", email, user.Provider)
		}
		if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(s.cfg.LocalAdmin.Password)) != nil {
			hash, hashErr := bcrypt.GenerateFromPassword([]byte(s.cfg.LocalAdmin.Password), bcrypt.DefaultCost)
			if hashErr != nil {
				return fmt.Errorf("hash local admin password: %w", hashErr)
			}
			if updateErr := s.store.UpdateUserPasswordHash(ctx, user.ID, string(hash)); updateErr != nil {
				return fmt.Errorf("sync local admin password: %w", updateErr)
			}
		}
		return s.store.AssignRoleByName(ctx, user.ID, "admin")
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(s.cfg.LocalAdmin.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash local admin password: %w", err)
	}
	user := &model.User{
		Username:     fallbackUsername(s.cfg.LocalAdmin.Username, email),
		Email:        email,
		PasswordHash: string(hash),
		Provider:     "local",
		Status:       "active",
	}
	if err := s.store.CreateUser(ctx, user); err != nil {
		return err
	}
	return s.store.AssignRoleByName(ctx, user.ID, "admin")
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	return s.LoginWithSource(ctx, req, "")
}

func (s *Service) LoginWithSource(ctx context.Context, req LoginRequest, sourceIP string) (LoginResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	keys := s.limiter.loginKeys(sourceIP, email)
	if s.limiter.blocked(keys) {
		return LoginResponse{}, rateLimitedError()
	}

	result, err := s.login(ctx, req, email)
	if err != nil {
		if IsUnauthenticated(err) && s.limiter.recordFailure(keys, s.limiter.login) {
			return LoginResponse{}, rateLimitedError()
		}
		return LoginResponse{}, err
	}
	s.limiter.reset(keys)
	return result, nil
}

func (s *Service) login(ctx context.Context, req LoginRequest, email string) (LoginResponse, error) {
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return LoginResponse{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid email or password", http.StatusUnauthorized)
		}
		return LoginResponse{}, err
	}
	if user.Status != "active" {
		return LoginResponse{}, apperrors.New(apperrors.CodePermissionDenied, "user is disabled", http.StatusForbidden)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return LoginResponse{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid email or password", http.StatusUnauthorized)
	}
	if s.mfaRequired() {
		purpose := mfaPurposeVerify
		if user.MFASecret == "" {
			purpose = mfaPurposeEnroll
		}
		return s.issueMFAChallenge(*user, purpose)
	}
	return s.issueForUser(*user, false)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (LoginResponse, error) {
	principal, err := s.tokens.Parse(refreshToken, TokenTypeRefresh)
	if err != nil {
		return LoginResponse{}, err
	}
	user, err := s.store.GetUserByID(ctx, principal.UserID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return LoginResponse{}, apperrors.New(apperrors.CodeUnauthenticated, "user not found", http.StatusUnauthorized)
		}
		return LoginResponse{}, err
	}
	if user.Status != "active" {
		return LoginResponse{}, apperrors.New(apperrors.CodePermissionDenied, "user is disabled", http.StatusForbidden)
	}
	if s.cfg.MFAEnabled {
		if user.MFASecret == "" {
			return LoginResponse{}, apperrors.New(apperrors.CodeUnauthenticated, "mfa enrollment required", http.StatusUnauthorized)
		}
		if !principal.MFAVerified {
			return LoginResponse{}, apperrors.New(apperrors.CodeUnauthenticated, "mfa verification required", http.StatusUnauthorized)
		}
	}
	return s.issueForUser(*user, s.cfg.MFAEnabled && principal.MFAVerified && user.MFASecret != "")
}

func (s *Service) ParseAccessToken(token string) (Principal, error) {
	return s.tokens.Parse(token, TokenTypeAccess)
}

func (s *Service) AuthenticateAccessToken(ctx context.Context, token string) (Principal, error) {
	principal, err := s.tokens.Parse(token, TokenTypeAccess)
	if err != nil {
		return Principal{}, err
	}
	user, err := s.store.GetUserByID(ctx, principal.UserID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return Principal{}, apperrors.New(apperrors.CodeUnauthenticated, "user not found", http.StatusUnauthorized)
		}
		return Principal{}, err
	}
	if user.Status != "active" {
		return Principal{}, apperrors.New(apperrors.CodePermissionDenied, "user is disabled", http.StatusForbidden)
	}
	if s.cfg.MFAEnabled {
		if user.MFASecret == "" {
			return Principal{}, apperrors.New(apperrors.CodeUnauthenticated, "mfa enrollment required", http.StatusUnauthorized)
		}
		if !principal.MFAVerified {
			return Principal{}, apperrors.New(apperrors.CodeUnauthenticated, "mfa verification required", http.StatusUnauthorized)
		}
	}
	return principalForUser(*user, s.cfg.MFAEnabled && principal.MFAVerified && user.MFASecret != ""), nil
}

func (s *Service) SetupMFA(ctx context.Context, mfaToken string) (MFASetupResponse, error) {
	challenge, err := s.tokens.ParseMFAChallenge(mfaToken)
	if err != nil {
		return MFASetupResponse{}, err
	}
	if challenge.Purpose != mfaPurposeEnroll || challenge.Secret != "" {
		return MFASetupResponse{}, apperrors.New(apperrors.CodeInvalidArgument, "mfa setup is not required", http.StatusBadRequest)
	}
	user, err := s.activeUser(ctx, challenge.Principal.UserID)
	if err != nil {
		return MFASetupResponse{}, err
	}
	if user.MFASecret != "" {
		return MFASetupResponse{}, apperrors.New(apperrors.CodeConflict, "mfa is already enabled", http.StatusConflict)
	}
	return s.newMFASetup(*user)
}

func (s *Service) StartMFAEnrollment(ctx context.Context, userID int64) (MFASetupResponse, error) {
	user, err := s.activeUser(ctx, userID)
	if err != nil {
		return MFASetupResponse{}, err
	}
	if user.MFASecret != "" {
		return MFASetupResponse{}, apperrors.New(apperrors.CodeConflict, "mfa is already enabled", http.StatusConflict)
	}
	return s.newMFASetup(*user)
}

func (s *Service) VerifyMFA(ctx context.Context, req MFAVerifyRequest) (LoginResponse, error) {
	return s.VerifyMFAWithSource(ctx, req, "")
}

func (s *Service) VerifyMFAWithSource(ctx context.Context, req MFAVerifyRequest, sourceIP string) (LoginResponse, error) {
	keys := s.limiter.mfaKeys(sourceIP, req.MFAToken)
	if s.limiter.blocked(keys) {
		return LoginResponse{}, rateLimitedError()
	}

	result, err := s.verifyMFAChallenge(ctx, 0, req)
	if err != nil {
		if IsUnauthenticated(err) && s.limiter.recordFailure(keys, s.limiter.mfa) {
			return LoginResponse{}, rateLimitedError()
		}
		return LoginResponse{}, err
	}
	s.limiter.reset(keys)
	return result, nil
}

func (s *Service) EnableMFA(ctx context.Context, userID int64, req MFAVerifyRequest) (LoginResponse, error) {
	return s.verifyMFAChallenge(ctx, userID, req)
}

func (s *Service) MFAStatus(ctx context.Context, userID int64) (MFAStatus, error) {
	user, err := s.activeUser(ctx, userID)
	if err != nil {
		return MFAStatus{}, err
	}
	return MFAStatus{
		Enabled:  user.MFASecret != "",
		Required: s.cfg.MFAEnabled,
	}, nil
}

func (s *Service) DisableMFA(ctx context.Context, userID int64, req MFADisableRequest) (LoginResponse, error) {
	if s.cfg.MFAEnabled {
		return LoginResponse{}, apperrors.New(apperrors.CodePermissionDenied, "mfa is required by server policy", http.StatusForbidden)
	}
	user, err := s.activeUser(ctx, userID)
	if err != nil {
		return LoginResponse{}, err
	}
	if user.MFASecret == "" {
		return LoginResponse{}, apperrors.New(apperrors.CodeConflict, "mfa is not enabled", http.StatusConflict)
	}
	if user.Provider != "local" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return LoginResponse{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid password or authentication code", http.StatusUnauthorized)
	}
	secret, err := s.secrets.Decrypt(user.MFASecret)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("decrypt user mfa secret: %w", err)
	}
	if !s.totp.Validate(secret, req.Code) {
		return LoginResponse{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid password or authentication code", http.StatusUnauthorized)
	}
	if err := s.store.UpdateUserMFASecret(ctx, user.ID, ""); err != nil {
		return LoginResponse{}, mapUserWriteError(err, "disable mfa failed")
	}
	user.MFASecret = ""
	user.MFAEnabled = false
	return s.issueForUser(*user, false)
}

func (s *Service) ListUsers(ctx context.Context) ([]model.User, error) {
	return s.store.ListUsers(ctx)
}

func (s *Service) ListRoles(ctx context.Context) ([]model.Role, error) {
	return s.store.ListRoles(ctx)
}

func (s *Service) CreateLocalUser(ctx context.Context, req CreateUserRequest) (model.User, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if username == "" || email == "" || req.Password == "" {
		return model.User{}, apperrors.New(apperrors.CodeInvalidArgument, "username, email and password are required", http.StatusBadRequest)
	}
	roles, err := s.normalizeRoles(ctx, req.Roles, true)
	if err != nil {
		return model.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, fmt.Errorf("hash user password: %w", err)
	}
	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Provider:     "local",
		Status:       "active",
	}
	if err := s.store.CreateUser(ctx, user); err != nil {
		return model.User{}, mapUserWriteError(err, "create user failed")
	}
	if err := s.store.ReplaceUserRoles(ctx, user.ID, roles); err != nil {
		return model.User{}, mapUserWriteError(err, "assign user roles failed")
	}
	created, err := s.store.GetUserByID(ctx, user.ID)
	if err != nil {
		return model.User{}, err
	}
	return *created, nil
}

func (s *Service) UpdateUserStatus(ctx context.Context, currentUserID, userID int64, req UpdateUserStatusRequest) (model.User, error) {
	status := strings.TrimSpace(strings.ToLower(req.Status))
	if status != "active" && status != "disabled" {
		return model.User{}, apperrors.New(apperrors.CodeInvalidArgument, "status must be active or disabled", http.StatusBadRequest)
	}
	if userID == currentUserID && status == "disabled" {
		return model.User{}, apperrors.New(apperrors.CodeInvalidArgument, "cannot disable current user", http.StatusBadRequest)
	}
	if err := s.store.UpdateUserStatus(ctx, userID, status); err != nil {
		return model.User{}, mapUserWriteError(err, "update user status failed")
	}
	updated, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return model.User{}, err
	}
	return *updated, nil
}

func (s *Service) ReplaceUserRoles(ctx context.Context, currentUserID, userID int64, req UpdateUserRolesRequest) (model.User, error) {
	if userID == currentUserID {
		return model.User{}, apperrors.New(apperrors.CodeInvalidArgument, "cannot update current user roles", http.StatusBadRequest)
	}
	roles, err := s.normalizeRoles(ctx, req.Roles, false)
	if err != nil {
		return model.User{}, err
	}
	if err := s.store.ReplaceUserRoles(ctx, userID, roles); err != nil {
		return model.User{}, mapUserWriteError(err, "replace user roles failed")
	}
	updated, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return model.User{}, err
	}
	return *updated, nil
}

func (s *Service) ResetUserMFA(ctx context.Context, currentUserID, userID int64, confirm bool) (model.User, error) {
	if !confirm {
		return model.User{}, apperrors.New(apperrors.CodeInvalidArgument, "confirm=true is required", http.StatusBadRequest)
	}
	if userID == currentUserID {
		return model.User{}, apperrors.New(apperrors.CodeInvalidArgument, "cannot reset current user mfa", http.StatusBadRequest)
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return model.User{}, mapUserWriteError(err, "reset user mfa failed")
	}
	if user.MFASecret == "" {
		return model.User{}, apperrors.New(apperrors.CodeConflict, "mfa is not enabled", http.StatusConflict)
	}
	if err := s.store.UpdateUserMFASecret(ctx, userID, ""); err != nil {
		return model.User{}, mapUserWriteError(err, "reset user mfa failed")
	}
	updated, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return model.User{}, err
	}
	return *updated, nil
}

func (s *Service) issueForUser(user model.User, mfaVerified bool) (LoginResponse, error) {
	principal := principalForUser(user, mfaVerified)
	accessToken, err := s.tokens.Issue(principal, TokenTypeAccess, s.cfg.AccessTokenTTL)
	if err != nil {
		return LoginResponse{}, err
	}
	refreshToken, err := s.tokens.Issue(principal, TokenTypeRefresh, s.cfg.RefreshTokenTTL)
	if err != nil {
		return LoginResponse{}, err
	}
	return LoginResponse{
		TokenPair: TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    int64(s.cfg.AccessTokenTTL.Seconds()),
		},
		User:        principal,
		MFARequired: false,
	}, nil
}

func (s *Service) issueMFAChallenge(user model.User, purpose string) (LoginResponse, error) {
	principal := principalForUser(user, false)
	token, err := s.tokens.IssueMFAChallenge(principal, purpose, "", s.cfg.MFAChallengeTTL)
	if err != nil {
		return LoginResponse{}, err
	}
	return LoginResponse{
		User:             principal,
		MFARequired:      true,
		MFASetupRequired: purpose == mfaPurposeEnroll,
		MFAToken:         token,
	}, nil
}

func (s *Service) newMFASetup(user model.User) (MFASetupResponse, error) {
	secret, err := s.totp.GenerateSecret()
	if err != nil {
		return MFASetupResponse{}, err
	}
	token, err := s.tokens.IssueMFAChallenge(principalForUser(user, false), mfaPurposeEnroll, secret, s.cfg.MFAChallengeTTL)
	if err != nil {
		return MFASetupResponse{}, err
	}
	return MFASetupResponse{
		Secret:          secret,
		ProvisioningURI: s.totp.ProvisioningURI(user.Email, secret),
		MFAToken:        token,
		ExpiresIn:       int64(s.cfg.MFAChallengeTTL.Seconds()),
	}, nil
}

func (s *Service) verifyMFAChallenge(ctx context.Context, expectedUserID int64, req MFAVerifyRequest) (LoginResponse, error) {
	challenge, err := s.tokens.ParseMFAChallenge(req.MFAToken)
	if err != nil {
		return LoginResponse{}, err
	}
	if expectedUserID > 0 && challenge.Principal.UserID != expectedUserID {
		return LoginResponse{}, apperrors.New(apperrors.CodePermissionDenied, "mfa challenge belongs to another user", http.StatusForbidden)
	}
	user, err := s.activeUser(ctx, challenge.Principal.UserID)
	if err != nil {
		return LoginResponse{}, err
	}

	storedSecret := user.MFASecret
	secret := ""
	switch challenge.Purpose {
	case mfaPurposeVerify:
		if storedSecret == "" {
			return LoginResponse{}, apperrors.New(apperrors.CodeUnauthenticated, "mfa enrollment required", http.StatusUnauthorized)
		}
		secret, err = s.secrets.Decrypt(storedSecret)
		if err != nil {
			return LoginResponse{}, fmt.Errorf("decrypt user mfa secret: %w", err)
		}
	case mfaPurposeEnroll:
		if challenge.Secret == "" {
			return LoginResponse{}, apperrors.New(apperrors.CodeInvalidArgument, "mfa setup must be completed first", http.StatusBadRequest)
		}
		if storedSecret != "" {
			return LoginResponse{}, apperrors.New(apperrors.CodeConflict, "mfa is already enabled", http.StatusConflict)
		}
		secret = challenge.Secret
	default:
		return LoginResponse{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid mfa challenge", http.StatusUnauthorized)
	}

	if !s.totp.Validate(secret, req.Code) {
		return LoginResponse{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid authentication code", http.StatusUnauthorized)
	}
	if challenge.Purpose == mfaPurposeVerify && !strings.HasPrefix(storedSecret, encryptedMFASecretPrefix) {
		encryptedSecret, err := s.secrets.Encrypt(secret)
		if err != nil {
			return LoginResponse{}, err
		}
		if err := s.store.UpdateUserMFASecret(ctx, user.ID, encryptedSecret); err != nil {
			return LoginResponse{}, mapUserWriteError(err, "upgrade mfa secret encryption failed")
		}
		user.MFASecret = encryptedSecret
	}
	if challenge.Purpose == mfaPurposeEnroll {
		encryptedSecret, err := s.secrets.Encrypt(secret)
		if err != nil {
			return LoginResponse{}, err
		}
		if err := s.store.UpdateUserMFASecret(ctx, user.ID, encryptedSecret); err != nil {
			return LoginResponse{}, mapUserWriteError(err, "enable mfa failed")
		}
		user.MFASecret = encryptedSecret
		user.MFAEnabled = true
	}
	return s.issueForUser(*user, true)
}

func (s *Service) activeUser(ctx context.Context, userID int64) (*model.User, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, apperrors.New(apperrors.CodeUnauthenticated, "user not found", http.StatusUnauthorized)
		}
		return nil, err
	}
	if user.Status != "active" {
		return nil, apperrors.New(apperrors.CodePermissionDenied, "user is disabled", http.StatusForbidden)
	}
	return user, nil
}

func (s *Service) mfaRequired() bool {
	return s.cfg.MFAEnabled
}

func rateLimitedError() error {
	return apperrors.New(apperrors.CodeRateLimited, "too many authentication attempts; try again later", http.StatusTooManyRequests)
}

func principalForUser(user model.User, mfaVerified bool) Principal {
	return Principal{
		UserID:      user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Roles:       user.Roles,
		MFAEnabled:  user.MFASecret != "",
		MFAVerified: mfaVerified,
	}
}

func fallbackUsername(username, email string) string {
	username = strings.TrimSpace(username)
	if username != "" {
		return username
	}
	if at := strings.Index(email, "@"); at > 0 {
		return email[:at]
	}
	return email
}

func (s *Service) normalizeRoles(ctx context.Context, input []string, defaultViewer bool) ([]string, error) {
	if len(input) == 0 && defaultViewer {
		input = []string{"viewer"}
	}
	if len(input) == 0 {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "at least one role is required", http.StatusBadRequest)
	}
	available, err := s.store.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]bool, len(available))
	order := make([]string, 0, len(available))
	for _, role := range available {
		allowed[role.Name] = true
		order = append(order, role.Name)
	}
	selected := make(map[string]bool, len(input))
	for _, role := range input {
		normalized := strings.TrimSpace(strings.ToLower(role))
		if normalized == "" {
			continue
		}
		if !allowed[normalized] {
			return nil, apperrors.New(apperrors.CodeInvalidArgument, fmt.Sprintf("unsupported role %q", normalized), http.StatusBadRequest)
		}
		selected[normalized] = true
	}
	if len(selected) == 0 {
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "at least one role is required", http.StatusBadRequest)
	}
	roles := make([]string, 0, len(selected))
	for _, role := range order {
		if selected[role] {
			roles = append(roles, role)
		}
	}
	return roles, nil
}

func mapUserWriteError(err error, message string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, store.ErrNotFound) {
		return apperrors.Wrap(err, apperrors.CodeNotFound, message, http.StatusNotFound)
	}
	if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed") {
		return apperrors.Wrap(err, apperrors.CodeConflict, message, http.StatusConflict)
	}
	return err
}
