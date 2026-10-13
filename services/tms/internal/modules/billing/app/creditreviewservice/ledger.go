package creditreviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type LedgerParams struct {
	fx.In

	Store   customerledgerports.CustomerLedgerProjectionRepository `name:"customerLedgerStore"`
	Reviews *Service
	Logger  *zap.Logger
}

// reviewingLedger is the receivables ledger every writer appends to. Once the
// entries commit, the customers they belong to are reviewed, so an invoice
// posting, a payment, a credit memo or a reversal moves a customer's credit
// standing without each of those paths having to remember to.
type reviewingLedger struct {
	customerledgerports.CustomerLedgerProjectionRepository
	reviews *Service
	l       *zap.Logger
}

func NewLedger(p LedgerParams) customerledgerports.CustomerLedgerProjectionRepository {
	return &reviewingLedger{
		CustomerLedgerProjectionRepository: p.Store,
		reviews:                            p.Reviews,
		l:                                  p.Logger.Named("credit-review.ledger"),
	}
}

func (r *reviewingLedger) AppendEntries(
	ctx context.Context,
	entries []*customerledger.CustomerLedgerEntry,
) error {
	if err := r.CustomerLedgerProjectionRepository.AppendEntries(ctx, entries); err != nil {
		return err
	}

	for tenant, customerIDs := range customersByTenant(entries) {
		kernelports.AfterCommit(ctx, func(committedCtx context.Context) {
			if err := r.reviews.Review(committedCtx, tenant, customerIDs); err != nil {
				r.l.Warn("credit review after a ledger posting failed; the nightly sweep will retry",
					zap.String("organizationId", tenant.OrgID.String()),
					zap.Error(err))
			}
		})
	}
	return nil
}

func customersByTenant(
	entries []*customerledger.CustomerLedgerEntry,
) map[pagination.TenantInfo][]pulid.ID {
	byTenant := make(map[pagination.TenantInfo][]pulid.ID, 1)
	for _, entry := range entries {
		if entry == nil || entry.CustomerID.IsNil() {
			continue
		}
		tenant := pagination.TenantInfo{OrgID: entry.OrganizationID, BuID: entry.BusinessUnitID}
		byTenant[tenant] = append(byTenant[tenant], entry.CustomerID)
	}
	return byTenant
}
