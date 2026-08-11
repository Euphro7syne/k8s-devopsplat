package audit

import (
	"context"

	"ops-platform/internal/store"
)

type Service struct {
	store store.AuditStore
}

func NewService(store store.AuditStore) *Service {
	return &Service{store: store}
}

func (s *Service) ListLogs(ctx context.Context, query store.AuditLogQuery) (ListLogsResponse, error) {
	items, err := s.store.ListAuditLogs(ctx, query)
	if err != nil {
		return ListLogsResponse{}, err
	}
	return ListLogsResponse{Items: items}, nil
}
