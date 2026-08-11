package logquery

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

func (h *Handler) Register(r gin.IRoutes, readMiddleware ...gin.HandlerFunc) {
	mw := append([]gin.HandlerFunc{}, readMiddleware...)
	r.GET("/namespaces/:namespace/pods/:pod/logs", append(mw, h.podLogs)...)
	r.GET("/logs", append(mw, h.logs)...)
}

func (h *Handler) podLogs(c *gin.Context) {
	query := h.queryFromContext(c)
	query.Namespace = c.Param("namespace")
	query.Pod = c.Param("pod")
	h.runQuery(c, query)
}

func (h *Handler) logs(c *gin.Context) {
	query := h.queryFromContext(c)
	query.Namespace = c.Query("namespace")
	query.Pod = c.Query("pod")
	h.runQuery(c, query)
}

func (h *Handler) runQuery(c *gin.Context, query Query) {
	result, err := h.service.Query(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) queryFromContext(c *gin.Context) Query {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "200"), 10, 64)
	if err != nil {
		limit = 200
	}
	previous := c.Query("previous") == "true"
	return Query{
		Container: c.Query("container"),
		From:      c.Query("from"),
		Keyword:   c.Query("keyword"),
		Level:     c.Query("level"),
		Limit:     limit,
		Previous:  previous,
	}
}

func invalidLogQuery(message string) error {
	return apperrors.New(apperrors.CodeInvalidArgument, message, http.StatusBadRequest)
}
