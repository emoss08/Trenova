package invoicedisputeservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type fakeDisputeDB struct{}

func (fakeDisputeDB) DB() *bun.DB                          { return nil }
func (fakeDisputeDB) DBForContext(context.Context) bun.IDB { return nil }

func (fakeDisputeDB) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}
func (fakeDisputeDB) HealthCheck(context.Context) error { return nil }
func (fakeDisputeDB) IsHealthy(context.Context) bool    { return true }
func (fakeDisputeDB) Close() error                      { return nil }

type disputeFixture struct {
	orgID, buID, userID pulid.ID
	tenantInfo          pagination.TenantInfo
	actor               *servicesports.RequestActor
	inv                 *invoice.Invoice
	repo                *mocks.MockInvoiceDisputeRepository
	invoiceRepo         *mocks.MockInvoiceRepository
	adjustmentRepo      *mocks.MockInvoiceAdjustmentRepository
	svc                 *Service
}

func newDisputeFixture(t *testing.T) *disputeFixture {
	t.Helper()
	f := &disputeFixture{
		orgID:  pulid.MustNew("org_"),
		buID:   pulid.MustNew("bu_"),
		userID: pulid.MustNew("usr_"),
	}
	f.tenantInfo = pagination.TenantInfo{OrgID: f.orgID, BuID: f.buID, UserID: f.userID}
	f.actor = testutil.NewSessionActor(f.userID, f.orgID, f.buID)
	f.inv = &invoice.Invoice{
		ID:               pulid.MustNew("inv_"),
		OrganizationID:   f.orgID,
		BusinessUnitID:   f.buID,
		CustomerID:       pulid.MustNew("cus_"),
		Number:           "INV-9",
		Status:           invoice.StatusPosted,
		BillType:         billingqueue.BillTypeInvoice,
		TotalAmount:      decimal.NewFromInt(100),
		TotalAmountMinor: 10000,
		DisputeStatus:    invoice.DisputeStatusNone,
	}
	f.repo = mocks.NewMockInvoiceDisputeRepository(t)
	f.invoiceRepo = mocks.NewMockInvoiceRepository(t)
	f.adjustmentRepo = mocks.NewMockInvoiceAdjustmentRepository(t)
	f.svc = &Service{
		l:              zap.NewNop(),
		db:             fakeDisputeDB{},
		repo:           f.repo,
		invoiceRepo:    f.invoiceRepo,
		adjustmentRepo: f.adjustmentRepo,
		auditService:   &mocks.NoopAuditService{},
		realtime:       &mocks.NoopRealtimeService{},
	}

	return f
}

func (f *disputeFixture) lockInvoice() {
	f.invoiceRepo.EXPECT().
		LockForUpdate(mock.Anything, repositories.GetInvoiceByIDRequest{ID: f.inv.ID, TenantInfo: f.tenantInfo}).
		Return(f.inv, nil).
		Once()
}

func (f *disputeFixture) expectFlag(status invoice.DisputeStatus) {
	f.invoiceRepo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(updated *invoice.Invoice) bool {
			return updated.ID == f.inv.ID && updated.DisputeStatus == status
		})).
		RunAndReturn(func(_ context.Context, updated *invoice.Invoice) (*invoice.Invoice, error) {
			return updated, nil
		}).
		Once()
}

func (f *disputeFixture) openRequest(amount string) *servicesports.OpenInvoiceDisputeRequest {
	return &servicesports.OpenInvoiceDisputeRequest{
		InvoiceID:      f.inv.ID,
		TenantInfo:     f.tenantInfo,
		ReasonCode:     invoice.DisputeReasonRateDiscrepancy,
		DisputedAmount: decimal.RequireFromString(amount),
		Notes:          " Rate on lane differs from contract ",
	}
}

func (f *disputeFixture) openCase() *invoice.InvoiceDispute {
	return &invoice.InvoiceDispute{
		ID:                  pulid.MustNew("idsp_"),
		OrganizationID:      f.orgID,
		BusinessUnitID:      f.buID,
		InvoiceID:           f.inv.ID,
		CustomerID:          f.inv.CustomerID,
		Status:              invoice.DisputeCaseStatusOpen,
		ReasonCode:          invoice.DisputeReasonRateDiscrepancy,
		DisputedAmount:      decimal.NewFromInt(40),
		DisputedAmountMinor: 4000,
		OpenedByID:          f.userID,
		OpenedAt:            1_700_000_000,
	}
}

func TestOpenDisputeFlagsTheInvoice(t *testing.T) {
	t.Parallel()

	f := newDisputeFixture(t)
	f.lockInvoice()
	f.repo.EXPECT().
		GetOpenByInvoiceID(mock.Anything, repositories.GetOpenInvoiceDisputeRequest{InvoiceID: f.inv.ID, TenantInfo: f.tenantInfo}).
		Return(nil, errortypes.NewNotFoundError("none")).
		Once()
	f.repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(entity *invoice.InvoiceDispute) bool {
			return entity.InvoiceID == f.inv.ID &&
				entity.CustomerID == f.inv.CustomerID &&
				entity.Status == invoice.DisputeCaseStatusOpen &&
				entity.DisputedAmountMinor == 4050 &&
				entity.Notes == "Rate on lane differs from contract" &&
				entity.OpenedByID == f.userID
		})).
		RunAndReturn(func(_ context.Context, entity *invoice.InvoiceDispute) (*invoice.InvoiceDispute, error) {
			entity.ID = pulid.MustNew("idsp_")
			return entity, nil
		}).
		Once()
	f.expectFlag(invoice.DisputeStatusDisputed)

	created, err := f.svc.Open(t.Context(), f.openRequest("40.50"), f.actor)

	require.NoError(t, err)
	assert.True(t, created.IsOpen())
	assert.Equal(t, invoice.DisputeStatusDisputed, f.inv.DisputeStatus)
}

func TestOpenDisputeRejectsBadRequests(t *testing.T) {
	t.Parallel()

	f := newDisputeFixture(t)

	req := f.openRequest("10")
	req.ReasonCode = invoice.DisputeReasonCode("Vibes")
	_, err := f.svc.Open(t.Context(), req, f.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Choose a dispute reason")

	req = f.openRequest("0")
	_, err = f.svc.Open(t.Context(), req, f.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "greater than zero")

	_, err = f.svc.Open(t.Context(), f.openRequest("10"), &servicesports.RequestActor{})
	require.Error(t, err)
}

func TestOpenDisputeInvoiceRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(inv *invoice.Invoice)
		amount  string
		wantErr string
	}{
		{
			"voided",
			func(inv *invoice.Invoice) { inv.Status = invoice.StatusVoided },
			"10",
			"voided invoice",
		},
		{
			"draft",
			func(inv *invoice.Invoice) { inv.Status = invoice.StatusDraft },
			"10",
			"Only a posted invoice",
		},
		{
			"credit memo",
			func(inv *invoice.Invoice) { inv.BillType = billingqueue.BillTypeCreditMemo },
			"10",
			"invoices and debit memos",
		},
		{
			"nothing owed",
			func(inv *invoice.Invoice) { inv.AppliedAmountMinor = 10000 },
			"10",
			"no open balance",
		},
		{
			"over the balance",
			func(inv *invoice.Invoice) { inv.AppliedAmountMinor = 6000; inv.AppliedAmount = decimal.NewFromInt(60) },
			"40.01",
			"exceeds the open balance of 40.00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newDisputeFixture(t)
			tt.mutate(f.inv)
			f.lockInvoice()

			_, err := f.svc.Open(t.Context(), f.openRequest(tt.amount), f.actor)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestOpenDisputeRefusesASecondOpenCase(t *testing.T) {
	t.Parallel()

	f := newDisputeFixture(t)
	f.lockInvoice()
	f.repo.EXPECT().
		GetOpenByInvoiceID(mock.Anything, mock.Anything).
		Return(f.openCase(), nil).
		Once()

	_, err := f.svc.Open(t.Context(), f.openRequest("10"), f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already has an open dispute")
}

func TestResolveDisputeClearsTheFlag(t *testing.T) {
	t.Parallel()

	f := newDisputeFixture(t)
	f.inv.DisputeStatus = invoice.DisputeStatusDisputed
	dispute := f.openCase()
	f.repo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceDisputeByIDRequest{ID: dispute.ID, TenantInfo: f.tenantInfo}).
		Return(dispute, nil).
		Once()
	f.lockInvoice()
	f.repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(entity *invoice.InvoiceDispute) bool {
			return entity.Status == invoice.DisputeCaseStatusResolved &&
				entity.Resolution == invoice.DisputeResolutionInvoiceUpheld &&
				entity.ResolvedByID == f.userID &&
				entity.ResolvedAt != nil &&
				entity.ResolutionNotes == "Contract rate confirmed"
		})).
		RunAndReturn(func(_ context.Context, entity *invoice.InvoiceDispute) (*invoice.InvoiceDispute, error) {
			return entity, nil
		}).
		Once()
	f.expectFlag(invoice.DisputeStatusNone)

	resolved, err := f.svc.Resolve(t.Context(), &servicesports.ResolveInvoiceDisputeRequest{
		DisputeID:       dispute.ID,
		TenantInfo:      f.tenantInfo,
		Resolution:      invoice.DisputeResolutionInvoiceUpheld,
		ResolutionNotes: " Contract rate confirmed ",
	}, f.actor)

	require.NoError(t, err)
	assert.Equal(t, invoice.DisputeCaseStatusResolved, resolved.Status)
	assert.Equal(t, invoice.DisputeStatusNone, f.inv.DisputeStatus)
	f.adjustmentRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestResolveDisputeCreditIssuedNeedsAnExecutedAdjustmentOnTheInvoice(t *testing.T) {
	t.Parallel()

	adjustmentID := pulid.MustNew("iadj_")
	tests := []struct {
		name       string
		adjustment *invoiceadjustment.InvoiceAdjustment
		withID     bool
		wantErr    string
	}{
		{"no adjustment named", nil, false, "Name the executed adjustment"},
		{
			"adjustment on another invoice",
			&invoiceadjustment.InvoiceAdjustment{
				ID:                adjustmentID,
				OriginalInvoiceID: pulid.MustNew("inv_"),
				Status:            invoiceadjustment.StatusExecuted,
			},
			true,
			"does not belong to invoice INV-9",
		},
		{"adjustment not executed", nil, true, "has not executed yet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newDisputeFixture(t)
			dispute := f.openCase()
			f.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(dispute, nil).Once()
			f.lockInvoice()
			if tt.withID {
				adjustment := tt.adjustment
				if adjustment == nil {
					adjustment = &invoiceadjustment.InvoiceAdjustment{
						ID:                adjustmentID,
						OriginalInvoiceID: f.inv.ID,
						Status:            invoiceadjustment.StatusPendingApproval,
					}
				}
				f.adjustmentRepo.EXPECT().
					GetByID(mock.Anything, repositories.GetInvoiceAdjustmentRequest{ID: adjustmentID, TenantInfo: f.tenantInfo}).
					Return(adjustment, nil).
					Once()
			}

			req := &servicesports.ResolveInvoiceDisputeRequest{
				DisputeID:  dispute.ID,
				TenantInfo: f.tenantInfo,
				Resolution: invoice.DisputeResolutionCreditIssued,
			}
			if tt.withID {
				req.ResolutionAdjustmentID = adjustmentID
			}
			_, err := f.svc.Resolve(t.Context(), req, f.actor)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestResolveDisputeWrittenOffWithExecutedAdjustmentSucceeds(t *testing.T) {
	t.Parallel()

	f := newDisputeFixture(t)
	f.inv.DisputeStatus = invoice.DisputeStatusDisputed
	dispute := f.openCase()
	adjustmentID := pulid.MustNew("iadj_")
	f.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(dispute, nil).Once()
	f.lockInvoice()
	f.adjustmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&invoiceadjustment.InvoiceAdjustment{ID: adjustmentID, OriginalInvoiceID: f.inv.ID, Status: invoiceadjustment.StatusExecuted}, nil).
		Once()
	f.repo.EXPECT().Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoice.InvoiceDispute) (*invoice.InvoiceDispute, error) {
			return entity, nil
		}).
		Once()
	f.expectFlag(invoice.DisputeStatusNone)

	resolved, err := f.svc.Resolve(t.Context(), &servicesports.ResolveInvoiceDisputeRequest{
		DisputeID:              dispute.ID,
		TenantInfo:             f.tenantInfo,
		Resolution:             invoice.DisputeResolutionWrittenOff,
		ResolutionAdjustmentID: adjustmentID,
	}, f.actor)

	require.NoError(t, err)
	assert.Equal(t, adjustmentID, resolved.ResolutionAdjustmentID)
}

func TestResolveAndWithdrawRefuseAClosedCase(t *testing.T) {
	t.Parallel()

	f := newDisputeFixture(t)
	closed := f.openCase()
	closed.Status = invoice.DisputeCaseStatusWithdrawn
	f.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(closed, nil).Twice()

	_, err := f.svc.Resolve(t.Context(), &servicesports.ResolveInvoiceDisputeRequest{
		DisputeID:  closed.ID,
		TenantInfo: f.tenantInfo,
		Resolution: invoice.DisputeResolutionInvoiceUpheld,
	}, f.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Only an open dispute can be resolved")

	_, err = f.svc.Withdraw(t.Context(), &servicesports.WithdrawInvoiceDisputeRequest{
		DisputeID:  closed.ID,
		TenantInfo: f.tenantInfo,
	}, f.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Only an open dispute can be withdrawn")
}

func TestWithdrawDisputeClearsTheFlag(t *testing.T) {
	t.Parallel()

	f := newDisputeFixture(t)
	f.inv.DisputeStatus = invoice.DisputeStatusDisputed
	dispute := f.openCase()
	f.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(dispute, nil).Once()
	f.lockInvoice()
	f.repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(entity *invoice.InvoiceDispute) bool {
			return entity.Status == invoice.DisputeCaseStatusWithdrawn &&
				entity.ResolvedByID == f.userID &&
				entity.ResolutionNotes == "Customer paid in full"
		})).
		RunAndReturn(func(_ context.Context, entity *invoice.InvoiceDispute) (*invoice.InvoiceDispute, error) {
			return entity, nil
		}).
		Once()
	f.expectFlag(invoice.DisputeStatusNone)

	withdrawn, err := f.svc.Withdraw(t.Context(), &servicesports.WithdrawInvoiceDisputeRequest{
		DisputeID:  dispute.ID,
		TenantInfo: f.tenantInfo,
		Notes:      "Customer paid in full",
	}, f.actor)

	require.NoError(t, err)
	assert.Equal(t, invoice.DisputeCaseStatusWithdrawn, withdrawn.Status)
	assert.Equal(t, invoice.DisputeStatusNone, f.inv.DisputeStatus)
}
