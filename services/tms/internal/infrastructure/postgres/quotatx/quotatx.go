package quotatx

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/uptrace/bun"
)

func Run(
	ctx context.Context,
	db kernelports.DBConnection,
	guard platformplanports.QuotaGuard,
	fn func(context.Context) error,
	reqs ...platformplanports.QuotaRequest,
) error {
	if !quotaservice.Enforcing(guard) {
		return fn(ctx)
	}

	return db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if err := quotaservice.EnforceAll(txCtx, guard, reqs...); err != nil {
			return err
		}

		return fn(txCtx)
	})
}
