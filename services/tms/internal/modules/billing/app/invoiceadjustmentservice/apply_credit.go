package invoiceadjustmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type creditApplier interface {
	ApplyCreditMemo(
		ctx context.Context,
		req *customerpaymentports.ApplyCreditMemoRequest,
		actor *permissionports.RequestActor,
	) ([]*customerpayment.CreditMemoApplication, error)
}

// applyCreditToSource settles the invoice an adjustment corrected with the
// credit memo the adjustment raised, up to what the invoice still owes. Without
// it the invoice stayed fully open beside a credit memo carrying the same
// money, so aging, statements and late charges went on treating it as unpaid.
// Credit beyond the open balance stays on the memo as the customer's
// unapplied credit, which is what the preview already said would happen.
//
// It returns the source invoice as it now stands, because applying the credit
// moved its balance and version.
func (s *Service) applyCreditToSource(
	ctx context.Context,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	creditMemo *invoice.Invoice,
	source *invoice.Invoice,
	actor *permissionports.RequestActor,
) (*invoice.Invoice, error) {
	amount := min(creditMemo.CreditRemainingMinor(), source.OpenBalanceMinor())
	if s.credits == nil || amount <= 0 {
		return source, nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  adjustment.OrganizationID,
		BuID:   adjustment.BusinessUnitID,
		UserID: actor.UserID,
	}
	if _, err := s.credits.ApplyCreditMemo(ctx, &customerpaymentports.ApplyCreditMemoRequest{
		CreditMemoID:   creditMemo.ID,
		AccountingDate: creditMemo.InvoiceDate,
		Applications: []*customerpaymentports.CreditMemoApplicationInput{{
			InvoiceID:          source.ID,
			AppliedAmountMinor: amount,
		}},
		TenantInfo: tenantInfo,
	}, actor); err != nil {
		return nil, err
	}

	return s.repo.LockInvoiceForUpdate(ctx, invoiceadjustmentports.LockInvoiceAdjustmentRequest{
		InvoiceID:  source.ID,
		TenantInfo: tenantInfo,
	})
}
