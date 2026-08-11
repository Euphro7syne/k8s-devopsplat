package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"ops-platform/internal/config"
)

type Store struct {
	db  *sql.DB
	log *slog.Logger
}

func Open(ctx context.Context, cfg config.DatabaseConfig, log *slog.Logger) (*Store, error) {
	if cfg.Driver != "sqlite" && cfg.Driver != "sqlite3" {
		return nil, fmt.Errorf("sqlite store only supports sqlite driver, got %q", cfg.Driver)
	}
	if cfg.DSN == "" {
		return nil, fmt.Errorf("database dsn is required")
	}
	if err := ensureSQLiteDir(cfg.DSN); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 1
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
		return fmt.Errorf("ping sqlite: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func ensureSQLiteDir(dsn string) error {
	path := dsn
	path = strings.TrimPrefix(path, "file:")
	if i := strings.Index(path, "?"); i >= 0 {
		path = path[:i]
	}
	if path == "" || path == ":memory:" || path == "file::memory:" {
		return nil
	}
	if strings.HasPrefix(path, "mode=memory") {
		return nil
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create sqlite dir %q: %w", dir, err)
	}
	return nil
}
