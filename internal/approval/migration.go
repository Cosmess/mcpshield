package approval

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_create_approval.sql
var approvalMigration string

func ApplyMigration(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, approvalMigration); err != nil {
		return fmt.Errorf("apply approval migration: %w", err)
	}
	return nil
}
