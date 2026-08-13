package migrations

import "embed"

// Files contains ordered SQL migrations.
//
//go:embed *.sql
var Files embed.FS

// PostgresFiles contains ordered PostgreSQL SQL migrations.
//
//go:embed postgres/*.sql
var PostgresFiles embed.FS
