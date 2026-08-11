package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "ops-platform/internal/pkg/errors"
	"ops-platform/internal/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterPublic(r gin.IRoutes) {
	r.POST("/auth/login", h.login)
	r.POST("/auth/refresh", h.refresh)
}

func (h *Handler) RegisterProtected(r gin.IRoutes) {
	r.GET("/auth/profile", h.profile)
	r.POST("/auth/mfa/verify", h.mfaVerifyPlaceholder)
}

func (h *Handler) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeInvalidArgument, "invalid login request", http.StatusBadRequest))
		return
	}
	result, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeInvalidArgument, "invalid refresh request", http.StatusBadRequest))
		return
	}
	result, err := h.service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) profile(c *gin.Context) {
	principal, ok := PrincipalFromContext(c)
	if !ok {
		response.Error(c, apperrors.New(apperrors.CodeUnauthenticated, "unauthenticated", http.StatusUnauthorized))
		return
	}
	response.Success(c, principal)
}

func (h *Handler) mfaVerifyPlaceholder(c *gin.Context) {
	response.Success(c, gin.H{"enabled": false})
}
