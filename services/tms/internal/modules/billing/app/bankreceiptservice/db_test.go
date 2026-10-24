package bankreceiptservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

type passthroughDB struct{}

func (passthroughDB) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	return fn(ctx)
}
