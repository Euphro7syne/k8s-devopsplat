package main

import (
	"context"
	stderrors "errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ops-platform/internal/config"
	"ops-platform/internal/pkg/logger"
	"ops-platform/internal/server"
	"ops-platform/internal/store/sqlite"
)

func main() {
	configPath := flag.String("config", "configs/ops-server.example.yaml", "path to ops-server config")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	log := logger.New(cfg.Log)
	ctx := context.Background()

	store, err := sqlite.Open(ctx, cfg.Database, log)
	if err != nil {
		log.Error("open store failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Warn("close store failed", "error", err)
		}
	}()

	if cfg.Database.AutoMigrate {
		if err := store.Migrate(ctx); err != nil {
			log.Error("run migrations failed", "error", err)
			os.Exit(1)
		}
	}

	app := server.New(cfg, store, log)
	httpServer := &http.Server{
		Addr:         cfg.Server.Listen,
		Handler:      app.Handler(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("ops-server listening", "addr", cfg.Server.Listen)
		if err := httpServer.ListenAndServe(); err != nil && !stderrors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	case sig := <-stopCh:
		log.Info("shutdown signal received", "signal", sig.String())
	}

	shutdownTimeout := cfg.Server.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 10 * time.Second
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}
}
