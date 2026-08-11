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

func (s *Store) ListUsers(ctx context.Context) ([]model.User, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, username, email, password_hash, mfa_secret, provider, status, created_at
FROM users
ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	users := make([]model.User, 0)
	for rows.Next() {
		var user model.User
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.PasswordHash,
			&user.MFASecret,
			&user.Provider,
			&user.Status,
			&user.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close user rows: %w", err)
	}
	for i := range users {
		roles, err := s.ListUserRoles(ctx, users[i].ID)
		if err != nil {
			return nil, err
		}
		users[i].Roles = roles
		users[i].MFAEnabled = users[i].MFASecret != ""
	}
	return users, nil
}

func (s *Store) ListRoles(ctx context.Context) ([]model.Role, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name
FROM roles
ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	roles := make([]model.Role, 0)
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(&role.ID, &role.Name); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roles: %w", err)
	}
	return roles, nil
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

func (s *Store) UpdateUserStatus(ctx context.Context, userID int64, status string) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE users
SET status = ?
WHERE id = ?`, status, userID)
	if err != nil {
		return fmt.Errorf("update user status: %w", err)
	}
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *Store) UpdateUserMFASecret(ctx context.Context, userID int64, secret string) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE users
SET mfa_secret = ?
WHERE id = ?`, secret, userID)
	if err != nil {
		return fmt.Errorf("update user mfa secret: %w", err)
	}
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return store.ErrNotFound
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

func (s *Store) ReplaceUserRoles(ctx context.Context, userID int64, roleNames []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace roles: %w", err)
	}
	defer tx.Rollback()

	var userExists int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM users WHERE id = ?", userID).Scan(&userExists); err != nil {
		return fmt.Errorf("check user: %w", err)
	}
	if userExists == 0 {
		return store.ErrNotFound
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM user_roles WHERE user_id = ?", userID); err != nil {
		return fmt.Errorf("delete user roles: %w", err)
	}
	for _, roleName := range roleNames {
		result, err := tx.ExecContext(ctx, `
INSERT INTO user_roles(user_id, role_id)
SELECT ?, id FROM roles WHERE name = ?`, userID, roleName)
		if err != nil {
			return fmt.Errorf("replace role %q: %w", roleName, err)
		}
		affected, err := result.RowsAffected()
		if err == nil && affected == 0 {
			return store.ErrNotFound
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace roles: %w", err)
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
	user.MFAEnabled = user.MFASecret != ""
	return &user, nil
}
