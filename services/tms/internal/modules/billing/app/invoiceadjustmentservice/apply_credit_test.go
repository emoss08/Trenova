package invoiceadjustmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceadjustmentportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingCreditApplier struct {
	requests []*customerpaymentports.ApplyCreditMemoRequest
}

func (r *recordingCreditApplier) ApplyCreditMemo(
	_ context.Context,
	req *customerpaymentports.ApplyCreditMemoRequest,
	_ *permissionports.RequestActor,
) ([]*customerpayment.CreditMemoApplication, error) {
	r.requests = append(r.requests, req)
	return nil, nil
}

func creditFixture(sourceTotal, sourceApplied, creditTotal int64) (*invoiceadjustment.InvoiceAdjustment, *invoice.Invoice, *invoice.Invoice) {
	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	adjustment := &invoiceadjustment.InvoiceAdjustment{
		ID: pulid.MustNew("iadj_"), OrganizationID: orgID, BusinessUnitID: buID,
	}
	source := &invoice.Invoice{
		ID: pulid.MustNew("inv_"), OrganizationID: orgID, BusinessUnitID: buID,
		BillType: billingqueue.BillTypeInvoice, Status: invoice.StatusPosted,
		TotalAmountMinor: sourceTotal, AppliedAmountMinor: sourceApplied,
	}
	memo := &invoice.Invoice{
		ID: pulid.MustNew("inv_"), OrganizationID: orgID, BusinessUnitID: buID,
		BillType: billingqueue.BillTypeCreditMemo, Status: invoice.StatusPosted,
		TotalAmountMinor: -creditTotal, InvoiceDate: 1_700_000_000,
	}

	return adjustment, source, memo
}

func TestApplyCreditToSourceSettlesWhatTheInvoiceStillOwes(t *testing.T) {
	t.Parallel()

	adjustment, source, memo := creditFixture(10_000, 7_000, 5_000)
	refreshed := &invoice.Invoice{ID: source.ID, AppliedAmountMinor: 10_000}
	repo := invoiceadjustmentportsmock.NewMockInvoiceAdjustmentRepository(t)
	repo.EXPECT().LockInvoiceForUpdate(mock.Anything, mock.Anything).Return(refreshed, nil).Once()
	credits := &recordingCreditApplier{}
	svc := &Service{repo: repo, credits: credits}

	got, err := svc.applyCreditToSource(
		t.Context(), adjustment, memo, source,
		testutil.NewSessionActor(pulid.MustNew("usr_"), adjustment.OrganizationID, adjustment.BusinessUnitID),
	)

	require.NoError(t, err)
	assert.Same(t, refreshed, got, "the caller continues with the invoice as it now stands")
	require.Len(t, credits.requests, 1)
	req := credits.requests[0]
	assert.Equal(t, memo.ID, req.CreditMemoID)
	assert.Equal(t, memo.InvoiceDate, req.AccountingDate)
	require.Len(t, req.Applications, 1)
	assert.Equal(t, source.ID, req.Applications[0].InvoiceID)
	assert.Equal(t, int64(3_000), req.Applications[0].AppliedAmountMinor,
		"only the open balance is settled; the rest stays as unapplied credit")
}

func TestApplyCreditToSourceLeavesAPaidInvoiceAlone(t *testing.T) {
	t.Parallel()

	adjustment, source, memo := creditFixture(10_000, 10_000, 2_000)
	credits := &recordingCreditApplier{}
	svc := &Service{credits: credits}

	got, err := svc.applyCreditToSource(
		t.Context(), adjustment, memo, source,
		testutil.NewSessionActor(pulid.MustNew("usr_"), adjustment.OrganizationID, adjustment.BusinessUnitID),
	)

	require.NoError(t, err)
	assert.Same(t, source, got)
	assert.Empty(t, credits.requests)
}
