package invoicememoservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicepostingservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceservicetest"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func memoRequest(
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
	billType billingqueue.BillType,
) *invoiceports.CreateMemoRequest {
	return &invoiceports.CreateMemoRequest{
		TenantInfo: tenantInfo,
		CustomerID: customerID,
		BillType:   billType,
		Reason:     "Goodwill credit after a late delivery",
		Lines: []*invoiceports.CreateMemoLineInput{
			{Description: "Late delivery goodwill", Amount: decimal.NewFromInt(50)},
			{
				Description: "Detention waived",
				Amount:      decimal.RequireFromString("25.5"),
				Quantity:    decimal.NewFromInt(1),
			},
		},
	}
}

func TestValidateMemoRequest(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	customerID := pulid.MustNew("cus_")

	tests := []struct {
		name    string
		mutate  func(req *invoiceports.CreateMemoRequest)
		wantErr string
	}{
		{
			name:    "an ordinary invoice is not a memo",
			mutate:  func(req *invoiceports.CreateMemoRequest) { req.BillType = billingqueue.BillTypeInvoice },
			wantErr: "credit memo or a debit memo",
		},
		{
			name:    "missing customer",
			mutate:  func(req *invoiceports.CreateMemoRequest) { req.CustomerID = pulid.Nil },
			wantErr: "Customer is required",
		},
		{
			name:    "reason is required",
			mutate:  func(req *invoiceports.CreateMemoRequest) { req.Reason = "  " },
			wantErr: "Say why",
		},
		{
			name:    "reason is capped",
			mutate:  func(req *invoiceports.CreateMemoRequest) { req.Reason = strings.Repeat("x", maxMemoReasonLength+1) },
			wantErr: "at most",
		},
		{
			name:    "at least one line",
			mutate:  func(req *invoiceports.CreateMemoRequest) { req.Lines = nil },
			wantErr: "at least one line",
		},
		{
			name:    "amount must be positive",
			mutate:  func(req *invoiceports.CreateMemoRequest) { req.Lines[0].Amount = decimal.Zero },
			wantErr: "Amount must be greater than zero",
		},
		{
			name:    "quantity must be positive when set",
			mutate:  func(req *invoiceports.CreateMemoRequest) { req.Lines[1].Quantity = decimal.NewFromInt(-1) },
			wantErr: "Quantity must be greater than zero",
		},
		{
			name:    "blank description",
			mutate:  func(req *invoiceports.CreateMemoRequest) { req.Lines[0].Description = " " },
			wantErr: "Description is required",
		},
		{
			name:    "unknown memo kind",
			mutate:  func(req *invoiceports.CreateMemoRequest) { req.MemoKind = invoice.MemoKind("Bonus") },
			wantErr: "Invalid memo kind",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := memoRequest(tenantInfo, customerID, billingqueue.BillTypeCreditMemo)
			tt.mutate(req)
			multiErr := validateMemoRequest(req)
			require.NotNil(t, multiErr)
			assert.Contains(t, multiErr.Error(), tt.wantErr)
		})
	}

	t.Run("a valid request passes", func(t *testing.T) {
		t.Parallel()
		assert.Nil(
			t,
			validateMemoRequest(
				memoRequest(tenantInfo, customerID, billingqueue.BillTypeDebitMemo),
			),
		)
	})
}

func TestValidateMemoReference(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	cus := &customer.Customer{ID: pulid.MustNew("cus_")}
	referenceID := pulid.MustNew("inv_")

	t.Run("no reference is fine", func(t *testing.T) {
		t.Parallel()
		svc := &Service{
			l:    zap.NewNop(),
			repo: invoiceportsmock.NewMockInvoiceRepository(t),
			rec:  invoiceaction.NewRecorder(invoiceaction.RecorderParams{Logger: zap.NewNop()}),
		}
		req := memoRequest(tenantInfo, cus.ID, billingqueue.BillTypeCreditMemo)
		require.NoError(t, svc.validateMemoReference(t.Context(), req, cus))
	})

	t.Run("another customer's invoice is refused", func(t *testing.T) {
		t.Parallel()
		repo := invoiceportsmock.NewMockInvoiceRepository(t)
		repo.EXPECT().
			GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: referenceID, TenantInfo: tenantInfo}).
			Return(&invoice.Invoice{ID: referenceID, CustomerID: pulid.MustNew("cus_"), Status: invoice.StatusPosted}, nil).
			Once()
		svc := &Service{
			l:    zap.NewNop(),
			repo: repo,
			rec:  invoiceaction.NewRecorder(invoiceaction.RecorderParams{Logger: zap.NewNop()}),
		}
		req := memoRequest(tenantInfo, cus.ID, billingqueue.BillTypeCreditMemo)
		req.ReferenceInvoiceID = referenceID
		err := svc.validateMemoReference(t.Context(), req, cus)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "another customer")
	})

	t.Run("a draft cannot be referenced", func(t *testing.T) {
		t.Parallel()
		repo := invoiceportsmock.NewMockInvoiceRepository(t)
		repo.EXPECT().
			GetByID(mock.Anything, mock.Anything).
			Return(&invoice.Invoice{ID: referenceID, CustomerID: cus.ID, Status: invoice.StatusDraft}, nil).
			Once()
		svc := &Service{
			l:    zap.NewNop(),
			repo: repo,
			rec:  invoiceaction.NewRecorder(invoiceaction.RecorderParams{Logger: zap.NewNop()}),
		}
		req := memoRequest(tenantInfo, cus.ID, billingqueue.BillTypeCreditMemo)
		req.ReferenceInvoiceID = referenceID
		err := svc.validateMemoReference(t.Context(), req, cus)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "posted invoice")
	})
}

type memoFixture struct {
	orgID, buID, userID, customerID pulid.ID
	tenantInfo                      pagination.TenantInfo
	actor                           *permissionports.RequestActor
	customerRepo                    *customerportsmock.MockCustomerRepository
	billingRepo                     *tenantportsmock.MockBillingControlRepository
	queueRepo                       *billingqueueportsmock.MockBillingQueueRepository
	repo                            *invoiceportsmock.MockInvoiceRepository
	svc                             *Service
}

func newMemoFixture(t *testing.T, number string) *memoFixture {
	t.Helper()
	f := &memoFixture{
		orgID:      pulid.MustNew("org_"),
		buID:       pulid.MustNew("bu_"),
		userID:     pulid.MustNew("usr_"),
		customerID: pulid.MustNew("cus_"),
	}
	f.tenantInfo = pagination.TenantInfo{OrgID: f.orgID, BuID: f.buID, UserID: f.userID}
	f.actor = testutil.NewSessionActor(f.userID, f.orgID, f.buID)

	f.customerRepo = customerportsmock.NewMockCustomerRepository(t)
	f.customerRepo.EXPECT().
		GetByID(mock.Anything, customerports.GetCustomerByIDRequest{
			ID:         f.customerID,
			TenantInfo: f.tenantInfo,
			CustomerFilterOptions: customerports.CustomerFilterOptions{
				IncludeBillingProfile: true,
				IncludeState:          true,
			},
		}).
		Return(&customer.Customer{
			ID:             f.customerID,
			OrganizationID: f.orgID,
			BusinessUnitID: f.buID,
			Name:           "AMD",
			Code:           "AMD01",
			AddressLine1:   "1 Chip Way",
			City:           "Austin",
			PostalCode:     "78701",
			BillingProfile: &customer.CustomerBillingProfile{
				PaymentTerm: customer.PaymentTermNet15,
			},
		}, nil).
		Once()

	f.billingRepo = tenantportsmock.NewMockBillingControlRepository(t)
	f.billingRepo.EXPECT().
		GetByOrgID(mock.Anything, f.orgID).
		Return(&billingcontrol.BillingControl{DefaultPaymentTerm: billingcontrol.PaymentTermNet30}, nil).
		Once()

	f.queueRepo = billingqueueportsmock.NewMockBillingQueueRepository(t)
	f.queueRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(item *billingqueue.BillingQueueItem) bool {
			return item.BillToCustomerID == f.customerID &&
				item.ShipmentID.IsNil() && item.OrderID.IsNil() &&
				item.Status == billingqueue.StatusApproved &&
				item.Number == number
		})).
		RunAndReturn(func(_ context.Context, item *billingqueue.BillingQueueItem) (*billingqueue.BillingQueueItem, error) {
			item.ID = pulid.MustNew("bqi_")
			return item, nil
		}).
		Once()

	f.repo = invoiceportsmock.NewMockInvoiceRepository(t)
	f.svc = &Service{
		l:                zap.NewNop(),
		db:               invoiceservicetest.FakeTx{},
		repo:             f.repo,
		billingQueueRepo: f.queueRepo,
		customerRepo:     f.customerRepo,
		billingRepo:      f.billingRepo,
		validator: invoicevalidation.NewValidator(
			invoicevalidation.ValidatorParams{Logger: zap.NewNop()},
		),
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: number},
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{
				Logger:       zap.NewNop(),
				AuditService: &mocks.NoopAuditService{},
				Realtime:     &mocks.NoopRealtimeService{},
			},
		),
	}
	f.svc.posting = invoicepostingservice.New(invoicepostingservice.Params{
		Logger:            zap.NewNop(),
		DB:                invoiceservicetest.FakeTx{},
		Repo:              f.repo,
		BillingQueueRepo:  f.queueRepo,
		CustomerRepo:      f.customerRepo,
		BillingRepo:       f.billingRepo,
		Validator:         f.svc.validator,
		SequenceGenerator: testutil.TestSequenceGenerator{SingleValue: number},
		AuditService:      &mocks.NoopAuditService{},
		Realtime:          &mocks.NoopRealtimeService{},
	})

	return f
}

func TestCreateMemoBuildsASignedCreditMemoOnItsOwnQueueItem(t *testing.T) {
	t.Parallel()

	f := newMemoFixture(t, "CM-0001")
	f.repo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			entity.ID = pulid.MustNew("inv_")
			return entity, nil
		}).
		Once()

	req := memoRequest(f.tenantInfo, f.customerID, billingqueue.BillTypeCreditMemo)
	req.InvoiceDate = 1_700_000_000
	req.Memo = "  Approved by AR lead  "

	created, err := f.svc.CreateMemo(t.Context(), req, f.actor)

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, invoice.ScopeMemo, created.Scope)
	assert.Equal(t, invoice.StatusDraft, created.Status)
	assert.Equal(t, "CM-0001", created.Number)
	assert.Equal(t, invoice.MemoKindManual, created.MemoKind, "kind defaults to Manual")
	assert.Equal(t, "Goodwill credit after a late delivery", created.MemoReason)
	assert.Equal(t, "Approved by AR lead", created.Memo)
	assert.Equal(t, "AMD", created.BillToName)
	assert.Equal(t, "AMD01", created.BillToCode)
	assert.Equal(
		t,
		invoice.PaymentTermNet15,
		created.PaymentTerm,
		"the customer's term beats the tenant default",
	)
	require.NotNil(t, created.DueDate)
	assert.Equal(
		t,
		int64(1_700_000_000),
		*created.DueDate,
		"a credit memo is due on its invoice date",
	)
	require.Len(t, created.Lines, 2)
	assert.Equal(t, invoice.InvoiceLineTypeMemo, created.Lines[0].Type)
	assert.True(
		t,
		created.Lines[0].Amount.Equal(decimal.NewFromInt(-50)),
		"credit lines are negative",
	)
	assert.True(t, created.Lines[1].Amount.Equal(decimal.RequireFromString("-25.5")))
	assert.True(
		t,
		created.TotalAmount.Equal(decimal.RequireFromString("-75.5")),
		"got %s",
		created.TotalAmount,
	)
	assert.Equal(t, int64(-7550), created.TotalAmountMinor)
}

func TestCreateMemoDebitMemoIsPositiveAndDueByTerm(t *testing.T) {
	t.Parallel()

	f := newMemoFixture(t, "DM-0009")
	f.repo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			entity.ID = pulid.MustNew("inv_")
			return entity, nil
		}).
		Once()

	req := memoRequest(f.tenantInfo, f.customerID, billingqueue.BillTypeDebitMemo)
	req.InvoiceDate = 1_700_000_000
	req.MemoKind = invoice.MemoKindLateCharge

	created, err := f.svc.CreateMemo(t.Context(), req, f.actor)

	require.NoError(t, err)
	assert.Equal(t, "DM-0009", created.Number)
	assert.Equal(t, invoice.MemoKindLateCharge, created.MemoKind)
	assert.True(t, created.TotalAmount.Equal(decimal.RequireFromString("75.5")))
	require.NotNil(t, created.DueDate)
	assert.Equal(t, int64(1_700_000_000+15*86400), *created.DueDate, "Net15 from the invoice date")
}

func TestCreateMemoAutoPostGoesThroughPost(t *testing.T) {
	t.Parallel()

	f := newMemoFixture(t, "DM-0010")
	memoID := pulid.MustNew("inv_")
	f.repo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			entity.ID = memoID
			return entity, nil
		}).
		Once()
	// Post reloads the memo first; failing that load proves the post was
	// attempted without needing the whole ledger behind it.
	f.repo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: memoID, TenantInfo: f.tenantInfo}).
		Return(nil, errors.New("ledger unavailable")).
		Once()

	req := memoRequest(f.tenantInfo, f.customerID, billingqueue.BillTypeDebitMemo)
	req.AutoPost = true

	_, err := f.svc.CreateMemo(t.Context(), req, f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ledger unavailable")
}
