package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ops-platform/internal/auth"
	"ops-platform/internal/config"
	"ops-platform/internal/k8s"
	"ops-platform/internal/resources"
	"ops-platform/internal/store"
)

type Server struct {
	cfg         *config.Config
	store       store.Store
	log         *slog.Logger
	authService *auth.Service
	kubeClient  resources.KubernetesClient
	engine      *gin.Engine
}

func New(cfg *config.Config, store store.Store, log *slog.Logger) *Server {
	gin.SetMode(cfg.Server.Mode)

	authService := auth.NewService(store, cfg.Auth)
	if err := authService.BootstrapAdmin(context.Background()); err != nil && log != nil {
		log.Warn("bootstrap local admin failed", "error", err)
	}

	kubeClient, err := k8s.NewClientset(cfg.Kubernetes)
	if err != nil && log != nil {
		log.Warn("kubernetes client unavailable", "error", err)
	}

	s := &Server{
		cfg:         cfg,
		store:       store,
		log:         log,
		authService: authService,
		kubeClient:  kubeClient,
		engine:      gin.New(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.engine
}
