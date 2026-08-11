package audit

import "ops-platform/internal/model"

type ListLogsResponse struct {
	Items []model.AuditLog `json:"items"`
}
