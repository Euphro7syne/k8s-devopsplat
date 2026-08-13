package main

import (
	"context"
	"flag"
	"log"
	"os"

	"ops-platform/internal/config"
	"ops-platform/internal/pkg/logger"
	storefactory "ops-platform/internal/store/factory"
)

func main() {
	configPath := flag.String("config", "configs/ops-server.example.yaml", "path to ops-server config")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	appLogger := logger.New(cfg.Log)
	store, err := storefactory.Open(context.Background(), cfg.Database, appLogger)
	if err != nil {
		appLogger.Error("open store failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			appLogger.Warn("close store failed", "error", err)
		}
	}()

	if err := store.Migrate(context.Background()); err != nil {
		appLogger.Error("run migrations failed", "error", err)
		os.Exit(1)
	}
	appLogger.Info("migrations completed")
}
