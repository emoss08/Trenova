package bankreceiptservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/uptrace/bun"
)

type passthroughDB struct{}

func (passthroughDB) DB() *bun.DB                          { return nil }
func (passthroughDB) DBForContext(context.Context) bun.IDB { return nil }
func (passthroughDB) HealthCheck(context.Context) error    { return nil }
func (passthroughDB) IsHealthy(context.Context) bool       { return true }
func (passthroughDB) Close() error                         { return nil }

func (passthroughDB) WithTx(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}
