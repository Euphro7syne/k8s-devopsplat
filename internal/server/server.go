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
	cfg           *config.Config
	store         store.Store
	log           *slog.Logger
	authService   *auth.Service
	kubeClient    resources.KubernetesClient
	resourceCache *k8s.ResourceCache
	cacheCancel   context.CancelFunc
	engine        *gin.Engine
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
	if kubeClient != nil && cfg.Kubernetes.Cache.Enabled {
		cacheCtx, cancel := context.WithCancel(context.Background())
		s.cacheCancel = cancel
		s.resourceCache = k8s.NewResourceCache(kubeClient, cfg.Kubernetes.Cache.ResyncPeriod)
		go func() {
			if err := s.resourceCache.Start(cacheCtx, cfg.Kubernetes.Cache.SyncTimeout); err != nil {
				wasRunning := cacheCtx.Err() == nil
				cancel()
				if wasRunning && log != nil {
					log.Warn("kubernetes resource cache unavailable; falling back to direct API reads", "error", err)
				}
				return
			}
			if log != nil {
				log.Info("kubernetes resource cache ready")
			}
		}()
	}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.engine
}

func (s *Server) Close() {
	if s == nil {
		return
	}
	if s.cacheCancel != nil {
		s.cacheCancel()
	}
	if s.resourceCache != nil {
		s.resourceCache.Shutdown()
	}
}
