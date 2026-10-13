package customerledgerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// SumCustomerLedgerBalanceRequest totals the AR subledger as at a date. The
// ledger is append-only and carries no fiscal year of its own, so a customer
// balance is every entry up to the date and nothing else.
type SumCustomerLedgerBalanceRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	AsOfDate   int64                 `json:"asOfDate"`
}

// SumCustomerBalancesRequest totals each customer's subledger: what it owes
// on open invoices, less unapplied cash and credits on its account.
type SumCustomerBalancesRequest struct {
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	CustomerIDs []pulid.ID            `json:"customerIds"`
	// ExcludeNonRecourseFactored leaves out every entry for an invoice sold to
	// a non-recourse factor: the invoice itself and whatever was applied to
	// it. The customer owes that money to the factor, which carries the risk.
	ExcludeNonRecourseFactored bool `json:"excludeNonRecourseFactored"`
}

type CustomerLedgerProjectionRepository interface {
	AppendEntries(ctx context.Context, entries []*customerledger.CustomerLedgerEntry) error
	SumBalanceAsOf(ctx context.Context, req SumCustomerLedgerBalanceRequest) (int64, error)
	SumCustomerBalances(
		ctx context.Context,
		req SumCustomerBalancesRequest,
	) (map[pulid.ID]int64, error)
}
