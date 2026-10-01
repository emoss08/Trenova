package postgres

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
)

const enableAuditRetentionSQL = "SET LOCAL trenova.audit_retention = 'on'"

func DeleteUnderAuditRetention(
	ctx context.Context,
	conn *Connection,
	build func(tx bun.Tx) *bun.DeleteQuery,
) (int64, error) {
	var deleted int64
	err := conn.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if err := enableAuditRetention(ctx, tx); err != nil {
			return err
		}

		result, err := build(tx).Exec(ctx)
		if err != nil {
			return err
		}

		deleted, err = result.RowsAffected()
		return err
	})
	if err != nil {
		return 0, err
	}

	return deleted, nil
}

func enableAuditRetention(ctx context.Context, tx bun.Tx) error {
	if tx.Dialect().Name() != dialect.PG {
		return nil
	}

	if _, err := tx.ExecContext(ctx, enableAuditRetentionSQL); err != nil {
		return fmt.Errorf("enable audit retention for this transaction: %w", err)
	}

	return nil
}
