package factory

import (
	"context"
	"testing"

	"ops-platform/internal/config"
)

func TestOpenRejectsUnsupportedDriver(t *testing.T) {
	_, err := Open(context.Background(), config.DatabaseConfig{
		Driver: "mysql",
		DSN:    "placeholder",
	}, nil)
	if err == nil {
		t.Fatalf("expected unsupported driver error")
	}
}
