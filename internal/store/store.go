package store

import (
	"context"
	"errors"

	"ops-platform/internal/model"
)

var ErrNotFound = errors.New("store: not found")

type Store interface {
	HealthStore
	MigrationStore
	AuthStore
	ClusterStore
	AuditStore
	Close() error
}

type HealthStore interface {
	Ping(ctx context.Context) error
}

type MigrationStore interface {
	Migrate(ctx context.Context) error
}

type AuditStore interface {
	CreateAuditLog(ctx context.Context, log *model.AuditLog) error
	ListAuditLogs(ctx context.Context, query AuditLogQuery) ([]model.AuditLog, error)
}

type AuditLogQuery struct {
	UserID       *int64
	Action       string
	ResourceName string
	Namespace    string
	From         string
	To           string
	Limit        int
}

type AuthStore interface {
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserByID(ctx context.Context, id int64) (*model.User, error)
	CreateUser(ctx context.Context, user *model.User) error
	AssignRoleByName(ctx context.Context, userID int64, roleName string) error
	ListUserRoles(ctx context.Context, userID int64) ([]string, error)
}

type ClusterStore interface {
	ListClusters(ctx context.Context) ([]model.Cluster, error)
}
