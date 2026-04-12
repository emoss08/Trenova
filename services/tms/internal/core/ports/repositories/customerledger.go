package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customerledger"
)

type CustomerLedgerProjectionRepository interface {
	AppendEntries(ctx context.Context, entries []*customerledger.Entry) error
}
