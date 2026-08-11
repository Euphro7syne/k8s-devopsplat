package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"ops-platform/internal/model"
	"ops-platform/internal/store"
)

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return s.getUser(ctx, "email = ?", email)
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	return s.getUser(ctx, "id = ?", id)
}

func (s *Store) CreateUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return nil
	}
	result, err := s.db.ExecContext(ctx, `
INSERT INTO users(username, email, password_hash, mfa_secret, provider, status)
VALUES (?, ?, ?, ?, ?, ?)`,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.MFASecret,
		user.Provider,
		user.Status,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	id, err := result.LastInsertId()
	if err == nil {
		user.ID = id
	}
	return nil
}

func (s *Store) AssignRoleByName(ctx context.Context, userID int64, roleName string) error {
	result, err := s.db.ExecContext(ctx, `
INSERT OR IGNORE INTO user_roles(user_id, role_id)
SELECT ?, id FROM roles WHERE name = ?`, userID, roleName)
	if err != nil {
		return fmt.Errorf("assign role %q: %w", roleName, err)
	}
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		var exists int
		if scanErr := s.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM roles WHERE name = ?", roleName).Scan(&exists); scanErr != nil {
			return fmt.Errorf("check role %q: %w", roleName, scanErr)
		}
		if exists == 0 {
			return store.ErrNotFound
		}
	}
	return nil
}

func (s *Store) ListUserRoles(ctx context.Context, userID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT r.name
FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = ?
ORDER BY r.name`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user roles: %w", err)
	}
	defer rows.Close()

	roles := make([]string, 0)
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roles: %w", err)
	}
	return roles, nil
}

func (s *Store) getUser(ctx context.Context, where string, args ...any) (*model.User, error) {
	query := `
SELECT id, username, email, password_hash, mfa_secret, provider, status, created_at
FROM users
WHERE ` + where

	var user model.User
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.MFASecret,
		&user.Provider,
		&user.Status,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	roles, err := s.ListUserRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles
	return &user, nil
}
