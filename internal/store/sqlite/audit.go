package sqlite

import (
	"context"
	"fmt"
	"strings"

	"ops-platform/internal/model"
	"ops-platform/internal/store"
)

func (s *Store) CreateAuditLog(ctx context.Context, log *model.AuditLog) error {
	if log == nil {
		return nil
	}
	result, err := s.db.ExecContext(ctx, `
INSERT INTO audit_logs(user_id, action, resource_type, resource_name, namespace, request_body, ip)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		log.UserID,
		log.Action,
		log.ResourceType,
		log.ResourceName,
		log.Namespace,
		log.RequestBody,
		log.IP,
	)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	id, err := result.LastInsertId()
	if err == nil {
		log.ID = id
	}
	return nil
}

func (s *Store) ListAuditLogs(ctx context.Context, query store.AuditLogQuery) ([]model.AuditLog, error) {
	where := []string{"1=1"}
	args := make([]any, 0)
	if query.UserID != nil {
		where = append(where, "user_id = ?")
		args = append(args, *query.UserID)
	}
	if query.Action != "" {
		where = append(where, "action = ?")
		args = append(args, query.Action)
	}
	if query.ResourceName != "" {
		where = append(where, "resource_name LIKE ?")
		args = append(args, "%"+query.ResourceName+"%")
	}
	if query.Namespace != "" {
		where = append(where, "namespace = ?")
		args = append(args, query.Namespace)
	}
	if query.From != "" {
		where = append(where, "created_at >= ?")
		args = append(args, query.From)
	}
	if query.To != "" {
		where = append(where, "created_at <= ?")
		args = append(args, query.To)
	}
	limit := query.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, `
SELECT id, user_id, action, resource_type, resource_name, namespace, request_body, ip, created_at
FROM audit_logs
WHERE `+strings.Join(where, " AND ")+`
ORDER BY created_at DESC, id DESC
LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	logs := make([]model.AuditLog, 0)
	for rows.Next() {
		var item model.AuditLog
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Action,
			&item.ResourceType,
			&item.ResourceName,
			&item.Namespace,
			&item.RequestBody,
			&item.IP,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		logs = append(logs, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit logs: %w", err)
	}
	return logs, nil
}
