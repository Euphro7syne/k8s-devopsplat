package factory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"ops-platform/internal/config"
	"ops-platform/internal/store"
	"ops-platform/internal/store/postgres"
	"ops-platform/internal/store/sqlite"
)

func Open(ctx context.Context, cfg config.DatabaseConfig, log *slog.Logger) (store.Store, error) {
	cfg.Driver = strings.ToLower(strings.TrimSpace(cfg.Driver))
	switch cfg.Driver {
	case "sqlite", "sqlite3":
		return sqlite.Open(ctx, cfg, log)
	case "postgres", "postgresql":
		return postgres.Open(ctx, cfg, log)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", cfg.Driver)
	}
}
