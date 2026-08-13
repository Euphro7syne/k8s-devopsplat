package postgres

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
	if err := s.db.QueryRowContext(ctx, `
INSERT INTO audit_logs(user_id, action, resource_type, resource_name, namespace, request_body, ip)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id`,
		log.UserID,
		log.Action,
		log.ResourceType,
		log.ResourceName,
		log.Namespace,
		log.RequestBody,
		log.IP,
	).Scan(&log.ID); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (s *Store) ListAuditLogs(ctx context.Context, query store.AuditLogQuery) ([]model.AuditLog, error) {
	where := []string{"1=1"}
	args := make([]any, 0)
	addArgument := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.UserID != nil {
		where = append(where, "user_id = "+addArgument(*query.UserID))
	}
	if query.Action != "" {
		where = append(where, "action = "+addArgument(query.Action))
	}
	if query.ResourceName != "" {
		where = append(where, "resource_name LIKE "+addArgument("%"+query.ResourceName+"%"))
	}
	if query.Namespace != "" {
		where = append(where, "namespace = "+addArgument(query.Namespace))
	}
	if query.From != "" {
		where = append(where, "created_at >= "+addArgument(query.From))
	}
	if query.To != "" {
		where = append(where, "created_at <= "+addArgument(query.To))
	}
	limit := query.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	limitPlaceholder := addArgument(limit)

	rows, err := s.db.QueryContext(ctx, `
SELECT id, user_id, action, resource_type, resource_name, namespace, request_body, ip, created_at
FROM audit_logs
WHERE `+strings.Join(where, " AND ")+`
ORDER BY created_at DESC, id DESC
LIMIT `+limitPlaceholder, args...)
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
