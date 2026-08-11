package workload

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

func (h *Handler) Register(r gin.IRoutes, writeMiddleware ...gin.HandlerFunc) {
	mw := append([]gin.HandlerFunc{}, writeMiddleware...)
	r.DELETE("/namespaces/:namespace/pods/:pod", append(mw, h.deletePod)...)
	r.POST("/namespaces/:namespace/deployments/:name/scale", append(mw, h.scaleDeployment)...)
	r.POST("/namespaces/:namespace/deployments/:name/restart", append(mw, h.restartDeployment)...)
}

func (h *Handler) deletePod(c *gin.Context) {
	result, err := h.service.DeletePod(c.Request.Context(), c.Param("namespace"), c.Param("pod"), c.Query("confirm") == "true")
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) scaleDeployment(c *gin.Context) {
	var req ScaleDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeInvalidArgument, "invalid scale request", http.StatusBadRequest))
		return
	}
	result, err := h.service.ScaleDeployment(c.Request.Context(), c.Param("namespace"), c.Param("name"), req.Replicas)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) restartDeployment(c *gin.Context) {
	result, err := h.service.RestartDeployment(c.Request.Context(), c.Param("namespace"), c.Param("name"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}
