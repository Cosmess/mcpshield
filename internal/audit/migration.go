package audit

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_create_audit.sql
var auditMigration string

func ApplyMigration(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, auditMigration); err != nil {
		return fmt.Errorf("apply audit migration: %w", err)
	}
	return nil
}
