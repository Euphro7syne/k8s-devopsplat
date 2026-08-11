package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ops-platform/internal/config"
	apperrors "ops-platform/internal/pkg/errors"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type Principal struct {
	UserID   int64    `json:"user_id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

type Claims struct {
	Subject   string   `json:"sub"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	Type      string   `json:"type"`
	Issuer    string   `json:"iss"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type TokenManager struct {
	issuer  string
	secret  []byte
	nowFunc func() time.Time
}

func NewTokenManager(cfg config.AuthConfig) *TokenManager {
	secret := []byte(cfg.JWTSecret)
	if len(secret) == 0 {
		secret = []byte("change-me-placeholder")
	}
	return &TokenManager{
		issuer:  cfg.JWTIssuer,
		secret:  secret,
		nowFunc: time.Now,
	}
}

func (m *TokenManager) Issue(principal Principal, tokenType string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		return "", fmt.Errorf("token ttl must be positive")
	}
	now := m.nowFunc().UTC()
	claims := Claims{
		Subject:   strconv.FormatInt(principal.UserID, 10),
		Username:  principal.Username,
		Email:     principal.Email,
		Roles:     principal.Roles,
		Type:      tokenType,
		Issuer:    m.issuer,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	}
	return m.sign(claims)
}

func (m *TokenManager) Parse(token, expectedType string) (Principal, error) {
	claims, err := m.parseClaims(token)
	if err != nil {
		return Principal{}, err
	}
	if claims.Type != expectedType {
		return Principal{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid token type", 401)
	}
	if claims.Issuer != m.issuer {
		return Principal{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid token issuer", 401)
	}
	if claims.ExpiresAt <= m.nowFunc().UTC().Unix() {
		return Principal{}, apperrors.New(apperrors.CodeUnauthenticated, "token expired", 401)
	}
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return Principal{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid token subject", 401)
	}
	return Principal{
		UserID:   userID,
		Username: claims.Username,
		Email:    claims.Email,
		Roles:    claims.Roles,
	}, nil
}

func (m *TokenManager) sign(claims Claims) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	headerRaw, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsRaw, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerRaw)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsRaw)
	signingInput := encodedHeader + "." + encodedClaims
	signature := m.signature(signingInput)
	return signingInput + "." + signature, nil
}

func (m *TokenManager) parseClaims(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid token", 401)
	}

	expected := m.signature(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return Claims{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid token signature", 401)
	}

	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid token header", 401)
	}
	var header map[string]string
	if err := json.Unmarshal(headerRaw, &header); err != nil {
		return Claims{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid token header", 401)
	}
	if header["alg"] != "HS256" {
		return Claims{}, apperrors.New(apperrors.CodeUnauthenticated, "unsupported token algorithm", 401)
	}

	claimsRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid token claims", 401)
	}
	var claims Claims
	if err := json.Unmarshal(claimsRaw, &claims); err != nil {
		return Claims{}, apperrors.New(apperrors.CodeUnauthenticated, "invalid token claims", 401)
	}
	if claims.Subject == "" {
		return Claims{}, apperrors.New(apperrors.CodeUnauthenticated, "missing token subject", 401)
	}
	if claims.Type == "" {
		return Claims{}, apperrors.New(apperrors.CodeUnauthenticated, "missing token type", 401)
	}
	return claims, nil
}

func (m *TokenManager) signature(input string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func BearerToken(header string) (string, error) {
	if header == "" {
		return "", apperrors.New(apperrors.CodeUnauthenticated, "authorization header is required", 401)
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", apperrors.New(apperrors.CodeUnauthenticated, "authorization header must be bearer token", 401)
	}
	if strings.TrimSpace(parts[1]) == "" {
		return "", apperrors.New(apperrors.CodeUnauthenticated, "token is required", 401)
	}
	return parts[1], nil
}

func IsUnauthenticated(err error) bool {
	var appErr *apperrors.AppError
	return errors.As(err, &appErr) && appErr.Code == apperrors.CodeUnauthenticated
}
