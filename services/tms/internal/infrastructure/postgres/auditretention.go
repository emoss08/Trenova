package postgres

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
)

const enableAuditRetentionSQL = "SET LOCAL trenova.audit_retention = 'on'"

func EnableAuditRetention(ctx context.Context, tx bun.Tx) error {
	if tx.Dialect().Name() != dialect.PG {
		return nil
	}

	if _, err := tx.ExecContext(ctx, enableAuditRetentionSQL); err != nil {
		return fmt.Errorf("enable audit retention for this transaction: %w", err)
	}

	return nil
}
