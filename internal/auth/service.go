package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"ops-platform/internal/config"
	"ops-platform/internal/model"
	apperrors "ops-platform/internal/pkg/errors"
	"ops-platform/internal/store"
)

type Service struct {
	store  store.AuthStore
	cfg    config.AuthConfig
	tokens *TokenManager
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	TokenPair
	User Principal `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
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
	return &Service{
		store:  store,
		cfg:    cfg,
		tokens: NewTokenManager(cfg),
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
	if _, err := s.store.GetUserByEmail(ctx, email); err == nil {
		return nil
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
	email := strings.TrimSpace(strings.ToLower(req.Email))
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
	return s.issueForUser(*user)
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
	return s.issueForUser(*user)
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
	return Principal{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Roles:    user.Roles,
	}, nil
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

func (s *Service) issueForUser(user model.User) (LoginResponse, error) {
	principal := Principal{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Roles:    user.Roles,
	}
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
		User: principal,
	}, nil
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
