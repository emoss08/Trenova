package invoiceservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type postFixture struct {
	tenant     pagination.TenantInfo
	userID     pulid.ID
	arID       pulid.ID
	revenueID  pulid.ID
	periodID   pulid.ID
	invoice    *invoice.Invoice
	leg        *shipment.Shipment
	item       *billingqueue.BillingQueueItem
	repo       *mocks.MockInvoiceRepository
	shipments  *mocks.MockShipmentRepository
	queue      *mocks.MockBillingQueueRepository
	journal    *mocks.MockJournalPostingRepository
	accounting *mocks.MockAccountingControlRepository
	svc        *Service
}

// newPostFixture is a draft invoice for one shipment, under an organization
// that books invoices to the ledger as they post.
func newPostFixture(t *testing.T) *postFixture {
	t.Helper()

	f := &postFixture{
		tenant:    pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		userID:    pulid.MustNew("usr_"),
		arID:      pulid.MustNew("gla_"),
		revenueID: pulid.MustNew("gla_"),
		periodID:  pulid.MustNew("fp_"),
	}
	f.leg = &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		ProNumber:      "PRO-88",
		Status:         shipment.StatusReadyToInvoice,
	}
	f.item = &billingqueue.BillingQueueItem{
		ID:             pulid.MustNew("bqi_"),
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		Number:         "INV-3001",
		Status:         billingqueue.StatusApproved,
	}
	f.invoice = &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		OrganizationID:     f.tenant.OrgID,
		BusinessUnitID:     f.tenant.BuID,
		CustomerID:         pulid.MustNew("cus_"),
		ShipmentID:         f.leg.ID,
		BillingQueueItemID: f.item.ID,
		Number:             "INV-3001",
		Scope:              invoice.ScopeShipment,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		InvoiceDate:        1_700_000_000,
		BillToName:         "Acme Logistics",
		Status:             invoice.StatusDraft,
		BillType:           billingqueue.BillTypeInvoice,
		SubtotalAmount:     decimal.RequireFromString("1250.00"),
		TotalAmount:        decimal.RequireFromString("1250.00"),
		TotalAmountMinor:   125000,
		Lines: []*invoice.InvoiceLine{{
			ShipmentID:  f.leg.ID,
			LineNumber:  1,
			Type:        invoice.InvoiceLineTypeFreight,
			Description: "Linehaul",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   decimal.RequireFromString("1250.00"),
			Amount:      decimal.RequireFromString("1250.00"),
			AmountMinor: 125000,
		}},
	}

	f.repo = mocks.NewMockInvoiceRepository(t)
	f.repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, repositories.GetInvoiceByIDRequest) (*invoice.Invoice, error) {
			copied := *f.invoice
			return &copied, nil
		}).Maybe()

	f.shipments = mocks.NewMockShipmentRepository(t)
	f.shipments.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, *repositories.GetShipmentByIDRequest) (*shipment.Shipment, error) {
			copied := *f.leg
			return &copied, nil
		}).Maybe()

	f.queue = mocks.NewMockBillingQueueRepository(t)
	f.queue.EXPECT().ListActiveInvoiceItemsByShipmentIDs(mock.Anything, mock.Anything).
		Return(nil, nil).Maybe()
	f.queue.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(
			context.Context,
			*repositories.GetBillingQueueItemByIDRequest,
		) (*billingqueue.BillingQueueItem, error) {
			copied := *f.item
			return &copied, nil
		}).Maybe()

	f.accounting = mocks.NewMockAccountingControlRepository(t)
	f.accounting.EXPECT().GetByOrgID(mock.Anything, f.tenant.OrgID).Return(&tenant.AccountingControl{
		AccountingBasis:          tenant.AccountingBasisAccrual,
		RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost,
		JournalPostingMode:       tenant.JournalPostingModeAutomatic,
		AutoPostSourceEvents: []tenant.JournalSourceEventType{
			tenant.JournalSourceEventInvoicePosted,
		},
		ReconciliationMode:      tenant.ReconciliationModeDisabled,
		DefaultRevenueAccountID: f.revenueID,
		DefaultARAccountID:      f.arID,
	}, nil).Maybe()

	periods := mocks.NewMockFiscalPeriodRepository(t)
	periods.EXPECT().GetPeriodByDate(mock.Anything, mock.Anything).Return(&fiscalperiod.FiscalPeriod{
		ID:           f.periodID,
		FiscalYearID: pulid.MustNew("fy_"),
		Status:       fiscalperiod.StatusOpen,
	}, nil).Maybe()

	f.journal = mocks.NewMockJournalPostingRepository(t)
	f.svc = &Service{
		l:                 zap.NewNop(),
		db:                fakeInvoiceDB{},
		repo:              f.repo,
		shipmentRepo:      f.shipments,
		billingQueueRepo:  f.queue,
		accountingRepo:    f.accounting,
		journalRepo:       f.journal,
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		auditService:      &mocks.NoopAuditService{},
		realtime:          &mocks.NoopRealtimeService{},
		validator: &Validator{
			l:                zap.NewNop(),
			validator:        validationframework.NewTenantedValidatorBuilder[*invoice.Invoice]().Build(),
			accountingRepo:   f.accounting,
			fiscalPeriodRepo: periods,
		},
	}

	return f
}

func (f *postFixture) request() *servicesports.PostInvoiceRequest {
	return &servicesports.PostInvoiceRequest{InvoiceID: f.invoice.ID, TenantInfo: f.tenant}
}

func (f *postFixture) actor() *servicesports.RequestActor {
	return testutil.NewSessionActor(f.userID, f.tenant.OrgID, f.tenant.BuID)
}

// The preview is Post's own plan: the invoice it posts, the shipment it
// invoices, the queue item it settles and the journal lines it books, all
// without a write or a journal number.
func TestPreviewPost_PlansWhatPostWouldWrite(t *testing.T) {
	t.Parallel()

	f := newPostFixture(t)

	preview, err := f.svc.PreviewPost(t.Context(), f.request(), f.actor())
	require.NoError(t, err)

	require.NoError(t, preview.Refusal)
	assert.Equal(t, invoice.StatusDraft, preview.Before.Status)
	assert.Equal(t, invoice.StatusPosted, preview.After.Status)
	require.NotNil(t, preview.After.PostedAt)

	require.Len(t, preview.Legs, 1)
	assert.Equal(t, shipment.StatusReadyToInvoice, preview.Legs[0].Before.Status)
	assert.Equal(t, shipment.StatusInvoiced, preview.Legs[0].After.Status)

	require.NotNil(t, preview.QueueAfter)
	assert.Equal(t, billingqueue.StatusApproved, preview.QueueBefore.Status)
	assert.Equal(t, billingqueue.StatusPosted, preview.QueueAfter.Status)

	require.NotNil(t, preview.Journal)
	assert.Equal(t, f.periodID, preview.Journal.FiscalPeriodID)
	require.Len(t, preview.Journal.Lines, 2)
	assert.Equal(t, f.arID, preview.Journal.Lines[0].GLAccountID)
	assert.Equal(t, int64(125000), preview.Journal.Lines[0].DebitMinor)
	assert.Equal(t, f.revenueID, preview.Journal.Lines[1].GLAccountID)
	assert.Equal(t, int64(125000), preview.Journal.Lines[1].CreditMinor)
	assert.Empty(t, preview.AccountingSync, "no accounting connection is configured")
}

// Posting books what the preview showed: the same accounts and amounts.
func TestPreviewPost_AgreesWithThePostItPlans(t *testing.T) {
	t.Parallel()

	f := newPostFixture(t)
	preview, err := f.svc.PreviewPost(t.Context(), f.request(), f.actor())
	require.NoError(t, err)

	var booked *repositories.CreateJournalPostingParams
	f.journal.EXPECT().CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params repositories.CreateJournalPostingParams) error {
			booked = &params
			return nil
		}).Once()
	f.repo.EXPECT().Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			return entity, nil
		}).Once()
	f.shipments.EXPECT().UpdateDerivedState(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
			return entity, nil
		}).Once()
	f.queue.EXPECT().Update(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			entity *billingqueue.BillingQueueItem,
		) (*billingqueue.BillingQueueItem, error) {
			return entity, nil
		}).Once()
	f.queue.EXPECT().MarkPostedForInvoice(mock.Anything, mock.Anything).Return(0, nil).Once()

	posted, err := f.svc.Post(t.Context(), f.request(), f.actor())
	require.NoError(t, err)

	assert.Equal(t, preview.After.Status, posted.Status)
	require.NotNil(t, booked)
	require.Len(t, booked.Lines, len(preview.Journal.Lines))
	for idx, line := range booked.Lines {
		assert.Equal(t, preview.Journal.Lines[idx].GLAccountID, line.GLAccountID)
		assert.Equal(t, preview.Journal.Lines[idx].DebitMinor, line.DebitAmount)
		assert.Equal(t, preview.Journal.Lines[idx].CreditMinor, line.CreditAmount)
	}
}

// A post the validator would refuse is previewed as a refusal, in the
// validator's words, rather than as a write.
func TestPreviewPost_ReportsWhyPostWouldRefuse(t *testing.T) {
	t.Parallel()

	f := newPostFixture(t)
	f.invoice.Status = invoice.StatusVoided

	preview, err := f.svc.PreviewPost(t.Context(), f.request(), f.actor())
	require.NoError(t, err)

	require.Error(t, preview.Refusal)
	assert.Contains(t, preview.Refusal.Error(), "Voided invoices cannot be posted")
	assert.Nil(t, preview.Journal)
	assert.Empty(t, preview.Legs)
}

func TestPreviewApprovalInvoice_PlansTheDraftApprovalWouldSave(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	customerID := pulid.MustNew("cus_")
	item := &billingqueue.BillingQueueItem{
		ID:             pulid.MustNew("bqi_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ShipmentID:     pulid.MustNew("shp_"),
		Status:         billingqueue.StatusInReview,
		BillType:       billingqueue.BillTypeInvoice,
		Number:         "INV-1001",
	}
	item.Shipment = &shipment.Shipment{
		ID:                  item.ShipmentID,
		OrganizationID:      orgID,
		BusinessUnitID:      buID,
		CustomerID:          customerID,
		ProNumber:           "PRO123",
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(100)),
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(100)),
	}

	repo := mocks.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByBillingQueueItemID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("invoice not found")).Once()
	queue := mocks.NewMockBillingQueueRepository(t)
	queue.EXPECT().GetByID(mock.Anything, mock.MatchedBy(
		func(req *repositories.GetBillingQueueItemByIDRequest) bool {
			return req.ItemID == item.ID && req.ExpandShipmentDetails
		},
	)).Return(item, nil).Once()
	customers := mocks.NewMockCustomerRepository(t)
	customers.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&customer.Customer{
		ID:             customerID,
		Name:           "Acme Logistics",
		BillingProfile: &customer.CustomerBillingProfile{AutoBill: true, BillingCurrency: "USD"},
	}, nil).Once()
	billing := mocks.NewMockBillingControlRepository(t)
	billing.EXPECT().GetByOrgID(mock.Anything, orgID).
		Return(nil, errortypes.NewNotFoundError("no billing control")).Twice()

	svc := &Service{
		l:                zap.NewNop(),
		repo:             repo,
		billingQueueRepo: queue,
		shipmentRepo:     mocks.NewMockShipmentRepository(t),
		customerRepo:     customers,
		billingRepo:      billing,
		validator: &Validator{
			validator: validationframework.NewTenantedValidatorBuilder[*invoice.Invoice]().Build(),
		},
	}

	result, err := svc.PreviewApprovalInvoice(
		t.Context(),
		&servicesports.CreateInvoiceFromBillingQueueRequest{
			BillingQueueItemID: item.ID,
			TenantInfo:         tenantInfo,
			DeferToStatement:   true,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, result.Invoice)
	assert.Equal(t, "INV-1001", result.Invoice.Number)
	assert.Equal(t, invoice.StatusDraft, result.Invoice.Status)
	assert.True(t, result.Invoice.TotalAmount.Equal(decimal.NewFromInt(100)))
	assert.True(t, result.Invoice.ID.IsNil(), "nothing was saved")
	assert.True(t, result.AutoPost, "the customer's profile posts its invoices on its own")
}
