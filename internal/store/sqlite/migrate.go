package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"ops-platform/internal/store/migrations"
)

func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	entries, err := migrations.Files.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		applied, err := s.isMigrationApplied(ctx, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		raw, err := migrations.Files.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if err := s.applyMigration(ctx, name, string(raw)); err != nil {
			return err
		}
		if s.log != nil {
			s.log.Info("migration applied", "version", name)
		}
	}
	return nil
}

func (s *Store) isMigrationApplied(ctx context.Context, version string) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations WHERE version = ?", version).Scan(&count); err != nil {
		return false, fmt.Errorf("check migration %s: %w", version, err)
	}
	return count > 0, nil
}

func (s *Store) applyMigration(ctx context.Context, version, sqlText string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", version, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, sqlText); err != nil {
		return fmt.Errorf("apply migration %s: %w", version, err)
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations(version) VALUES (?)", version); err != nil {
		return fmt.Errorf("record migration %s: %w", version, err)
	}
	if err = tx.Commit(); err != nil {
		if err == sql.ErrTxDone {
			return nil
		}
		return fmt.Errorf("commit migration %s: %w", version, err)
	}
	return nil
}
