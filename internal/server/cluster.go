package server

import (
	"github.com/gin-gonic/gin"

	"ops-platform/internal/pkg/response"
)

func (s *Server) clusters(c *gin.Context) {
	clusters, err := s.store.ListClusters(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, clusters)
}
