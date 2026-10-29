package invoiceadjustmentdraftservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceadjustmentportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeAdjustmentDB struct {
	transactions int
}

func (f *fakeAdjustmentDB) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	f.transactions++
	return fn(ctx)
}

func TestSaveDraftRefusesARequestWithoutADraftOrAnInvoice(t *testing.T) {
	t.Parallel()

	svc := &Service{db: &fakeAdjustmentDB{}}
	_, err := svc.SaveDraft(t.Context(), &invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest{
		Kind: invoiceadjustment.KindCreditOnly,
	}, testutil.NewSessionActor(pulid.MustNew("usr_"), pulid.MustNew("org_"), pulid.MustNew("bu_")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Name the invoice")

	_, err = svc.PreviewSaveDraft(
		t.Context(),
		&invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest{
			InvoiceID: pulid.MustNew("inv_"),
			Kind:      invoiceadjustment.Kind("Refund"),
		},
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Adjustment kind is invalid")
}

func TestSaveDraftOfAnExistingDraftOnlyUpdatesItInOneTransaction(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	submitted := &invoiceadjustment.InvoiceAdjustment{
		ID:     pulid.MustNew("iadj_"),
		Status: invoiceadjustment.StatusPendingApproval,
	}
	repo := invoiceadjustmentportsmock.NewMockInvoiceAdjustmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, invoiceadjustmentports.GetInvoiceAdjustmentRequest{ID: submitted.ID, TenantInfo: tenantInfo}).
		Return(submitted, nil).
		Once()
	db := &fakeAdjustmentDB{}
	svc := &Service{db: db, repo: repo}

	_, err := svc.SaveDraft(t.Context(), &invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest{
		AdjustmentID: submitted.ID,
		Kind:         invoiceadjustment.KindCreditOnly,
		TenantInfo:   tenantInfo,
	}, testutil.NewSessionActor(pulid.MustNew("usr_"), tenantInfo.OrgID, tenantInfo.BuID))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Only draft adjustments may be updated")
	assert.Equal(t, 1, db.transactions)
}

func TestPreviewSaveDraftRefusesAnAdjustmentThatIsNoLongerADraft(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	executed := &invoiceadjustment.InvoiceAdjustment{
		ID:     pulid.MustNew("iadj_"),
		Status: invoiceadjustment.StatusExecuted,
	}
	repo := invoiceadjustmentportsmock.NewMockInvoiceAdjustmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, invoiceadjustmentports.GetInvoiceAdjustmentRequest{ID: executed.ID, TenantInfo: tenantInfo}).
		Return(executed, nil).
		Once()
	svc := &Service{db: &fakeAdjustmentDB{}, repo: repo}

	_, err := svc.PreviewSaveDraft(
		t.Context(),
		&invoiceadjustmentports.SaveInvoiceAdjustmentDraftRequest{
			AdjustmentID: executed.ID,
			Kind:         invoiceadjustment.KindCreditOnly,
			TenantInfo:   tenantInfo,
		},
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Only draft adjustments may be updated")
}
