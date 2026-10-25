package postgres

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const enableAuditRetentionSQL = "SET LOCAL trenova.audit_retention = 'on'"

type AuditRetentionDelete func(ctx context.Context, db *keel.DB) (int64, error)

func DeleteUnderAuditRetention(
	ctx context.Context,
	conn *Connection,
	del AuditRetentionDelete,
) (int64, error) {
	var deleted int64
	err := conn.Transact(ctx, kernelports.TxOptions{}, func(ctx context.Context) error {
		db := conn.Keel(ctx)
		if _, err := keel.Exec(ctx, db, enableAuditRetentionSQL); err != nil {
			return fmt.Errorf("enable audit retention for this transaction: %w", err)
		}

		var err error
		deleted, err = del(ctx, db)
		return err
	})
	if err != nil {
		return 0, err
	}

	return deleted, nil
}
