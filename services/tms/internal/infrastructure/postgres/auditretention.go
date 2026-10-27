package postgres

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

type AuditRetentionDelete func(ctx context.Context, db *keel.DB) (int64, error)

func DeleteUnderAuditRetention(
	ctx context.Context,
	conn *Connection,
	del AuditRetentionDelete,
) (int64, error) {
	var deleted int64
	err := conn.Transact(ctx, kernelports.TxOptions{
		Settings: []kernelports.TxSetting{{Name: "trenova.audit_retention", Value: "on"}},
	}, func(ctx context.Context) error {
		var err error
		deleted, err = del(ctx, conn.Keel(ctx))
		return err
	})
	if err != nil {
		return 0, err
	}

	return deleted, nil
}
