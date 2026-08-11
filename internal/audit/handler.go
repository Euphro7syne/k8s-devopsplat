package audit

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"ops-platform/internal/pkg/response"
	"ops-platform/internal/store"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(r gin.IRoutes, readMiddleware ...gin.HandlerFunc) {
	mw := append([]gin.HandlerFunc{}, readMiddleware...)
	r.GET("/audit/logs", append(mw, h.logs)...)
}

func (h *Handler) logs(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		limit = 100
	}
	var userID *int64
	if raw := c.Query("user_id"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			userID = &parsed
		}
	}
	result, err := h.service.ListLogs(c.Request.Context(), store.AuditLogQuery{
		UserID:       userID,
		Action:       c.Query("action"),
		ResourceName: c.Query("resource"),
		Namespace:    c.Query("namespace"),
		From:         c.Query("from"),
		To:           c.Query("to"),
		Limit:        limit,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}
