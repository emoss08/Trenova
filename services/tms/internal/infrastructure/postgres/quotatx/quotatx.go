package quotatx

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

func Run(
	ctx context.Context,
	db kernelports.Transactor,
	guard platformplanports.QuotaGuard,
	fn func(context.Context) error,
	reqs ...platformplanports.QuotaRequest,
) error {
	if !quotaservice.Enforcing(guard) {
		return fn(ctx)
	}

	return db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		if err := quotaservice.EnforceAll(txCtx, guard, reqs...); err != nil {
			return err
		}

		return fn(txCtx)
	})
}
