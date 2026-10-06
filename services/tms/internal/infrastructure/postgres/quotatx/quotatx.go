package quotatx

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/quotaservice"
	"github.com/uptrace/bun"
)

func Run(
	ctx context.Context,
	db ports.DBConnection,
	guard services.QuotaGuard,
	fn func(context.Context) error,
	reqs ...services.QuotaRequest,
) error {
	if !quotaservice.Enforcing(guard) {
		return fn(ctx)
	}

	return db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if err := quotaservice.EnforceAll(txCtx, guard, reqs...); err != nil {
			return err
		}

		return fn(txCtx)
	})
}
