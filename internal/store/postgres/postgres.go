package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"

	"ops-platform/internal/config"
)

type Store struct {
	db  *sql.DB
	log *slog.Logger
}

func Open(ctx context.Context, cfg config.DatabaseConfig, log *slog.Logger) (*Store, error) {
	if cfg.Driver != "postgres" && cfg.Driver != "postgresql" {
		return nil, fmt.Errorf("postgres store only supports postgres driver, got %q", cfg.Driver)
	}
	if cfg.DSN == "" {
		return nil, fmt.Errorf("database dsn is required")
	}

	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 10
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle < 0 {
		maxIdle = 0
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)

	store := &Store{db: db, log: log}
	if err := store.Ping(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Ping(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
