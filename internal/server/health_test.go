package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes/fake"

	"ops-platform/internal/config"
	"ops-platform/internal/k8s"
)

func TestHealthReportsResourceCacheStateWithoutFailingFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := fake.NewSimpleClientset()
	cache := k8s.NewResourceCache(client, 0)
	cacheCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		cache.Shutdown()
	})
	if err := cache.Start(cacheCtx, 3*time.Second); err != nil {
		t.Fatalf("start resource cache: %v", err)
	}
	tests := []struct {
		name       string
		server     *Server
		kubernetes string
		cache      string
	}{
		{
			name:       "kubernetes unavailable",
			server:     &Server{cfg: cacheHealthConfig(true)},
			kubernetes: "unavailable",
			cache:      "unavailable",
		},
		{
			name:       "cache disabled",
			server:     &Server{cfg: cacheHealthConfig(false), kubeClient: fake.NewSimpleClientset()},
			kubernetes: "configured",
			cache:      "disabled",
		},
		{
			name:       "cache not ready falls back",
			server:     &Server{cfg: cacheHealthConfig(true), kubeClient: fake.NewSimpleClientset()},
			kubernetes: "configured",
			cache:      "not_ready",
		},
		{
			name:       "cache ready",
			server:     &Server{cfg: cacheHealthConfig(true), kubeClient: client, resourceCache: cache},
			kubernetes: "configured",
			cache:      "ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.GET("/healthz", tt.server.healthz)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("health returned HTTP %d: %s", recorder.Code, recorder.Body.String())
			}
			var envelope struct {
				Data healthResponse `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode health response: %v", err)
			}
			if envelope.Data.Kubernetes != tt.kubernetes || envelope.Data.ResourceCache != tt.cache {
				t.Fatalf("unexpected health state: %#v", envelope.Data)
			}
		})
	}
}

func cacheHealthConfig(enabled bool) *config.Config {
	cfg := config.Default()
	cfg.Kubernetes.Cache.Enabled = enabled
	return &cfg
}
