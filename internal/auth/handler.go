package auth

import (
	"net/http"
	"strconv"

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
	r.POST("/auth/mfa/setup", h.mfaSetup)
}

func (h *Handler) RegisterMFAVerification(r gin.IRoutes) {
	r.POST("/auth/mfa/verify", h.mfaVerify)
}

func (h *Handler) RegisterProtected(r gin.IRoutes) {
	r.GET("/auth/profile", h.profile)
	r.GET("/auth/mfa/status", h.mfaStatus)
	r.POST("/auth/mfa/enrollment", h.mfaEnrollment)
	r.POST("/auth/mfa/enable", h.mfaEnable)
	r.POST("/auth/mfa/disable", h.mfaDisable)
}

func (h *Handler) RegisterAdmin(r gin.IRoutes, adminMiddleware gin.HandlerFunc) {
	r.GET("/roles", adminMiddleware, h.roles)
	r.GET("/users", adminMiddleware, h.users)
	r.POST("/users", adminMiddleware, h.createUser)
	r.PUT("/users/:id/status", adminMiddleware, h.updateUserStatus)
	r.PUT("/users/:id/roles", adminMiddleware, h.updateUserRoles)
	r.DELETE("/users/:id/mfa", adminMiddleware, h.resetUserMFA)
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

func (h *Handler) mfaSetup(c *gin.Context) {
	var req MFASetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeInvalidArgument, "invalid mfa setup request", http.StatusBadRequest))
		return
	}
	result, err := h.service.SetupMFA(c.Request.Context(), req.MFAToken)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) mfaVerify(c *gin.Context) {
	var req MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeInvalidArgument, "invalid mfa verification request", http.StatusBadRequest))
		return
	}
	result, err := h.service.VerifyMFA(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	WithPrincipal(c, result.User)
	response.Success(c, result)
}

func (h *Handler) mfaStatus(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, apperrors.New(apperrors.CodeUnauthenticated, "unauthenticated", http.StatusUnauthorized))
		return
	}
	result, err := h.service.MFAStatus(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) mfaEnrollment(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, apperrors.New(apperrors.CodeUnauthenticated, "unauthenticated", http.StatusUnauthorized))
		return
	}
	result, err := h.service.StartMFAEnrollment(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) mfaEnable(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, apperrors.New(apperrors.CodeUnauthenticated, "unauthenticated", http.StatusUnauthorized))
		return
	}
	var req MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeInvalidArgument, "invalid mfa enable request", http.StatusBadRequest))
		return
	}
	result, err := h.service.EnableMFA(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) mfaDisable(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, apperrors.New(apperrors.CodeUnauthenticated, "unauthenticated", http.StatusUnauthorized))
		return
	}
	var req MFADisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeInvalidArgument, "invalid mfa disable request", http.StatusBadRequest))
		return
	}
	result, err := h.service.DisableMFA(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) roles(c *gin.Context) {
	result, err := h.service.ListRoles(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) users(c *gin.Context) {
	result, err := h.service.ListUsers(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) createUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeInvalidArgument, "invalid create user request", http.StatusBadRequest))
		return
	}
	result, err := h.service.CreateLocalUser(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) updateUserStatus(c *gin.Context) {
	userID, ok := h.userIDParam(c)
	if !ok {
		return
	}
	var req UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeInvalidArgument, "invalid status request", http.StatusBadRequest))
		return
	}
	currentUserID, ok := currentUserID(c)
	if !ok {
		response.Error(c, apperrors.New(apperrors.CodeUnauthenticated, "unauthenticated", http.StatusUnauthorized))
		return
	}
	result, err := h.service.UpdateUserStatus(c.Request.Context(), currentUserID, userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) updateUserRoles(c *gin.Context) {
	userID, ok := h.userIDParam(c)
	if !ok {
		return
	}
	var req UpdateUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeInvalidArgument, "invalid roles request", http.StatusBadRequest))
		return
	}
	currentUserID, ok := currentUserID(c)
	if !ok {
		response.Error(c, apperrors.New(apperrors.CodeUnauthenticated, "unauthenticated", http.StatusUnauthorized))
		return
	}
	result, err := h.service.ReplaceUserRoles(c.Request.Context(), currentUserID, userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) resetUserMFA(c *gin.Context) {
	userID, ok := h.userIDParam(c)
	if !ok {
		return
	}
	currentID, ok := currentUserID(c)
	if !ok {
		response.Error(c, apperrors.New(apperrors.CodeUnauthenticated, "unauthenticated", http.StatusUnauthorized))
		return
	}
	result, err := h.service.ResetUserMFA(c.Request.Context(), currentID, userID, c.Query("confirm") == "true")
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) userIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, apperrors.New(apperrors.CodeInvalidArgument, "invalid user id", http.StatusBadRequest))
		return 0, false
	}
	return id, true
}

func currentUserID(c *gin.Context) (int64, bool) {
	principal, ok := PrincipalFromContext(c)
	if !ok {
		return 0, false
	}
	return principal.UserID, true
}
