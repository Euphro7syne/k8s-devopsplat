package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ops-platform/internal/auth"
	"ops-platform/internal/model"
	apperrors "ops-platform/internal/pkg/errors"
	"ops-platform/internal/pkg/response"
	"ops-platform/internal/pkg/sanitizer"
)

func (s *Server) recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		if s.log != nil {
			s.log.Error("panic recovered", "panic", recovered)
		}
		response.Error(c, apperrors.New(apperrors.CodeInternal, "internal server error", http.StatusInternalServerError))
	})
}

func (s *Server) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && s.originAllowed(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", strings.Join(s.cfg.Server.CORS.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(s.cfg.Server.CORS.AllowedHeaders, ", "))
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (s *Server) originAllowed(origin string) bool {
	for _, allowed := range s.cfg.Server.CORS.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := auth.BearerToken(c.GetHeader("Authorization"))
		if err != nil {
			response.Error(c, err)
			c.Abort()
			return
		}
		principal, err := s.authService.AuthenticateAccessToken(c.Request.Context(), token)
		if err != nil {
			response.Error(c, err)
			c.Abort()
			return
		}
		auth.WithPrincipal(c, principal)
		c.Next()
	}
}

func (s *Server) requireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := auth.PrincipalFromContext(c)
		if !ok {
			response.Error(c, apperrors.New(apperrors.CodeUnauthenticated, "unauthenticated", http.StatusUnauthorized))
			c.Abort()
			return
		}
		if !auth.HasAnyRole(principal, roles...) {
			response.Error(c, apperrors.New(apperrors.CodePermissionDenied, "permission denied", http.StatusForbidden))
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *Server) auditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		mutating := c.Request.Method == http.MethodPost ||
			c.Request.Method == http.MethodPut ||
			c.Request.Method == http.MethodPatch ||
			c.Request.Method == http.MethodDelete
		if !mutating {
			c.Next()
			return
		}

		startedAt := time.Now()
		requestBody := s.readRequestBodyForAudit(c)
		c.Next()

		if s.store == nil {
			return
		}
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		var userID *int64
		switch raw := c.GetString("user_id"); {
		case raw != "":
			if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
				userID = &parsed
			}
		default:
			if principal, ok := auth.PrincipalFromContext(c); ok {
				userID = &principal.UserID
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err := s.store.CreateAuditLog(ctx, &model.AuditLog{
			UserID:       userID,
			Action:       c.Request.Method,
			ResourceType: "http",
			ResourceName: path,
			Namespace:    c.Param("namespace"),
			RequestBody:  requestBody,
			IP:           c.ClientIP(),
			CreatedAt:    startedAt,
		})
		if err != nil && s.log != nil {
			s.log.Warn("write audit log failed", "error", err, "path", path)
		}
	}
}

func (s *Server) readRequestBodyForAudit(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		if s.log != nil {
			s.log.Warn("read audit request body failed", "error", err)
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(nil))
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))

	body := sanitizer.String(string(raw))
	if len(body) > 8192 {
		return body[:8192]
	}
	return body
}
