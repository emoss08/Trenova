package dbtest

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

type NopConnection struct{}

func (NopConnection) Keel(context.Context) *keel.DB     { return nil }
func (NopConnection) HealthCheck(context.Context) error { return nil }
func (NopConnection) IsHealthy(context.Context) bool    { return true }
func (NopConnection) Close() error                      { return nil }

func (NopConnection) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	return fn(ctx)
}

func (NopConnection) Savepoint(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
