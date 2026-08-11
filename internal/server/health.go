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
	Status   string `json:"status"`
	Database string `json:"database"`
	Time     string `json:"time"`
}

func (s *Server) healthz(c *gin.Context) {
	payload := healthResponse{
		Status:   "ok",
		Database: "ok",
		Time:     time.Now().UTC().Format(time.RFC3339),
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
