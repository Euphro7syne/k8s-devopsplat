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
