package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	apperrors "ops-platform/internal/pkg/errors"
	"ops-platform/internal/pkg/response"
)

type healthResponse struct {
	Status        string `json:"status"`
	Database      string `json:"database"`
	Kubernetes    string `json:"kubernetes"`
	ResourceCache string `json:"resource_cache"`
	Time          string `json:"time"`
}

func (s *Server) healthz(c *gin.Context) {
	payload := healthResponse{
		Status:        "ok",
		Database:      "ok",
		Kubernetes:    "configured",
		ResourceCache: "disabled",
		Time:          time.Now().UTC().Format(time.RFC3339),
	}
	if s.kubeClient == nil {
		payload.Kubernetes = "unavailable"
		payload.ResourceCache = "unavailable"
	} else if s.cfg != nil && s.cfg.Kubernetes.Cache.Enabled {
		payload.ResourceCache = "not_ready"
		if s.resourceCache != nil && s.resourceCache.Ready() {
			payload.ResourceCache = "ready"
		}
	}

	if s.store != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := s.store.Ping(ctx); err != nil {
			payload.Status = "degraded"
			payload.Database = "error"
			response.ErrorWithData(c, apperrors.Wrap(err, apperrors.CodeServiceUnavailable, "database unavailable", http.StatusServiceUnavailable), payload)
			return
		}
	}

	response.Success(c, payload)
}
