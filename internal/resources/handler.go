package resources

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

func (h *Handler) Register(r gin.IRoutes, readMiddleware, writeMiddleware gin.HandlerFunc) {
	readMW := []gin.HandlerFunc{readMiddleware}
	writeMW := []gin.HandlerFunc{writeMiddleware}

	r.GET("/overview", append(readMW, h.overview)...)
	r.GET("/namespaces", append(readMW, h.namespaces)...)
	r.GET("/nodes", append(readMW, h.nodes)...)
	r.GET("/resources/yaml", append(readMW, h.resourceYAML)...)
	r.PUT("/resources/yaml", append(writeMW, h.updateResourceYAML)...)
	r.GET("/namespaces/:namespace/pods", append(readMW, h.pods)...)
	r.GET("/namespaces/:namespace/pods/:pod", append(readMW, h.pod)...)
	r.GET("/namespaces/:namespace/pods/:pod/yaml", append(readMW, h.podYAML)...)
	r.GET("/namespaces/:namespace/events", append(readMW, h.events)...)
	r.GET("/namespaces/:namespace/deployments", append(readMW, h.deployments)...)
	r.GET("/namespaces/:namespace/deployments/:name", append(readMW, h.deployment)...)
	r.GET("/namespaces/:namespace/deployments/:name/yaml", append(readMW, h.deploymentYAML)...)
	r.GET("/namespaces/:namespace/statefulsets", append(readMW, h.statefulSets)...)
	r.GET("/namespaces/:namespace/daemonsets", append(readMW, h.daemonSets)...)
	r.GET("/namespaces/:namespace/replicasets", append(readMW, h.replicaSets)...)
	r.GET("/namespaces/:namespace/jobs", append(readMW, h.jobs)...)
	r.GET("/namespaces/:namespace/cronjobs", append(readMW, h.cronJobs)...)
	r.GET("/namespaces/:namespace/services", append(readMW, h.services)...)
	r.GET("/namespaces/:namespace/ingresses", append(readMW, h.ingresses)...)
	r.GET("/namespaces/:namespace/configmaps", append(readMW, h.configMaps)...)
	r.GET("/namespaces/:namespace/persistentvolumeclaims", append(readMW, h.pvcs)...)
	r.GET("/persistentvolumes", append(readMW, h.pvs)...)
	r.GET("/storageclasses", append(readMW, h.storageClasses)...)
}

func (h *Handler) overview(c *gin.Context) {
	result, err := h.service.Overview(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) namespaces(c *gin.Context) {
	result, err := h.service.ListNamespaces(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) nodes(c *gin.Context) {
	result, err := h.service.ListNodes(c.Request.Context(), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) pods(c *gin.Context) {
	result, err := h.service.ListPods(c.Request.Context(), c.Param("namespace"), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) pod(c *gin.Context) {
	result, err := h.service.GetPod(c.Request.Context(), c.Param("namespace"), c.Param("pod"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) podYAML(c *gin.Context) {
	result, err := h.service.PodYAML(c.Request.Context(), c.Param("namespace"), c.Param("pod"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"yaml": result})
}

func (h *Handler) events(c *gin.Context) {
	result, err := h.service.ListEvents(c.Request.Context(), c.Param("namespace"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) deployments(c *gin.Context) {
	result, err := h.service.ListDeployments(c.Request.Context(), c.Param("namespace"), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) deployment(c *gin.Context) {
	result, err := h.service.GetDeployment(c.Request.Context(), c.Param("namespace"), c.Param("name"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) deploymentYAML(c *gin.Context) {
	result, err := h.service.DeploymentYAML(c.Request.Context(), c.Param("namespace"), c.Param("name"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"yaml": result})
}

func (h *Handler) statefulSets(c *gin.Context) {
	result, err := h.service.ListStatefulSets(c.Request.Context(), c.Param("namespace"), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) daemonSets(c *gin.Context) {
	result, err := h.service.ListDaemonSets(c.Request.Context(), c.Param("namespace"), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) replicaSets(c *gin.Context) {
	result, err := h.service.ListReplicaSets(c.Request.Context(), c.Param("namespace"), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) jobs(c *gin.Context) {
	result, err := h.service.ListJobs(c.Request.Context(), c.Param("namespace"), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) cronJobs(c *gin.Context) {
	result, err := h.service.ListCronJobs(c.Request.Context(), c.Param("namespace"), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) services(c *gin.Context) {
	result, err := h.service.ListServices(c.Request.Context(), c.Param("namespace"), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) ingresses(c *gin.Context) {
	result, err := h.service.ListIngresses(c.Request.Context(), c.Param("namespace"), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) configMaps(c *gin.Context) {
	result, err := h.service.ListConfigMaps(c.Request.Context(), c.Param("namespace"), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) pvcs(c *gin.Context) {
	result, err := h.service.ListPVCs(c.Request.Context(), c.Param("namespace"), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) pvs(c *gin.Context) {
	result, err := h.service.ListPVs(c.Request.Context(), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) storageClasses(c *gin.Context) {
	result, err := h.service.ListStorageClasses(c.Request.Context(), parseListOptions(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) resourceYAML(c *gin.Context) {
	result, err := h.service.ResourceYAML(c.Request.Context(), c.Query("kind"), c.Query("namespace"), c.Query("name"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"yaml": result})
}

func (h *Handler) updateResourceYAML(c *gin.Context) {
	var req ResourceYAMLUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeInvalidArgument, "invalid yaml update request", http.StatusBadRequest))
		return
	}
	result, err := h.service.UpdateResourceYAML(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func parseListOptions(c *gin.Context) ListOptions {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil {
		pageSize = 20
	}
	return ListOptions{Page: page, PageSize: pageSize}
}

func requireConfirm(c *gin.Context) error {
	if c.Query("confirm") != "true" {
		return apperrors.New(apperrors.CodeInvalidArgument, "confirm=true is required", http.StatusBadRequest)
	}
	return nil
}
