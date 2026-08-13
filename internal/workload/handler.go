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
	r.POST("/namespaces/:namespace/pods/:pod/restart", append(mw, h.restartPod)...)
	r.POST("/namespaces/:namespace/deployments/:name/scale", append(mw, h.scaleDeployment)...)
	r.POST("/namespaces/:namespace/deployments/:name/restart", append(mw, h.restartDeployment)...)
	r.POST("/namespaces/:namespace/statefulsets/:name/scale", append(mw, h.scaleStatefulSet)...)
	r.POST("/namespaces/:namespace/statefulsets/:name/restart", append(mw, h.restartStatefulSet)...)
	r.POST("/namespaces/:namespace/daemonsets/:name/restart", append(mw, h.restartDaemonSet)...)
	r.POST("/namespaces/:namespace/cronjobs/:name/suspend", append(mw, h.suspendCronJob)...)
	r.POST("/namespaces/:namespace/cronjobs/:name/resume", append(mw, h.resumeCronJob)...)
}

func (h *Handler) deletePod(c *gin.Context) {
	result, err := h.service.DeletePod(c.Request.Context(), c.Param("namespace"), c.Param("pod"), c.Query("confirm") == "true")
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) restartPod(c *gin.Context) {
	result, err := h.service.RestartPod(c.Request.Context(), c.Param("namespace"), c.Param("pod"), c.Query("confirm") == "true")
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

func (h *Handler) scaleStatefulSet(c *gin.Context) {
	var req ScaleStatefulSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeInvalidArgument, "invalid scale request", http.StatusBadRequest))
		return
	}
	result, err := h.service.ScaleStatefulSet(c.Request.Context(), c.Param("namespace"), c.Param("name"), req.Replicas)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) restartStatefulSet(c *gin.Context) {
	result, err := h.service.RestartStatefulSet(c.Request.Context(), c.Param("namespace"), c.Param("name"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) restartDaemonSet(c *gin.Context) {
	result, err := h.service.RestartDaemonSet(c.Request.Context(), c.Param("namespace"), c.Param("name"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) suspendCronJob(c *gin.Context) {
	h.setCronJobSuspend(c, true)
}

func (h *Handler) resumeCronJob(c *gin.Context) {
	h.setCronJobSuspend(c, false)
}

func (h *Handler) setCronJobSuspend(c *gin.Context, suspend bool) {
	result, err := h.service.SetCronJobSuspend(
		c.Request.Context(),
		c.Param("namespace"),
		c.Param("name"),
		suspend,
		c.Query("confirm") == "true",
	)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}
