package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customerledger"
	"github.com/emoss08/trenova/pkg/pagination"
)

// SumCustomerLedgerBalanceRequest totals the AR subledger as at a date. The
// ledger is append-only and carries no fiscal year of its own, so a customer
// balance is every entry up to the date and nothing else.
type SumCustomerLedgerBalanceRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	AsOfDate   int64                 `json:"asOfDate"`
}

type CustomerLedgerProjectionRepository interface {
	AppendEntries(ctx context.Context, entries []*customerledger.CustomerLedgerEntry) error
	SumBalanceAsOf(ctx context.Context, req SumCustomerLedgerBalanceRequest) (int64, error)
}
