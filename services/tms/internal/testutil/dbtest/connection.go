package dbtest

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/uptrace/bun"
)

type NopConnection struct{}

func (NopConnection) DB() *bun.DB                          { return nil }
func (NopConnection) DBForContext(context.Context) bun.IDB { return nil }
func (NopConnection) HealthCheck(context.Context) error    { return nil }
func (NopConnection) IsHealthy(context.Context) bool       { return true }
func (NopConnection) Close() error                         { return nil }

func (NopConnection) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}
