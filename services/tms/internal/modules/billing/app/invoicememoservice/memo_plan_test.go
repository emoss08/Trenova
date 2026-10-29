package invoicememoservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/billingtest/invoiceservicetest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
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

func TestPreviewMemoBuildsTheMemoWithoutMintingANumberOrWriting(t *testing.T) {
	t.Parallel()

	orgID, buID, userID := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")
	customerID := pulid.MustNew("cus_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	customers := customerportsmock.NewMockCustomerRepository(t)
	customers.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&customer.Customer{
		ID:             customerID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Name:           "AMD",
		Code:           "AMD01",
		BillingProfile: &customer.CustomerBillingProfile{
			PaymentTerm: customer.PaymentTermNet15,
		},
	}, nil).Once()
	billing := tenantportsmock.NewMockBillingControlRepository(t)
	billing.EXPECT().GetByOrgID(mock.Anything, orgID).
		Return(&billingcontrol.BillingControl{DefaultPaymentTerm: billingcontrol.PaymentTermNet30}, nil).
		Once()

	svc := &Service{
		l:                zap.NewNop(),
		db:               invoiceservicetest.FakeTx{},
		repo:             invoiceportsmock.NewMockInvoiceRepository(t),
		billingQueueRepo: billingqueueportsmock.NewMockBillingQueueRepository(t),
		customerRepo:     customers,
		billingRepo:      billing,
		validator: invoicevalidation.NewValidator(
			invoicevalidation.ValidatorParams{Logger: zap.NewNop()},
		),
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "CM-9"},
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{
				Logger:       zap.NewNop(),
				AuditService: &mocks.NoopAuditService{},
				Realtime:     &mocks.NoopRealtimeService{},
			},
		),
	}

	req := memoRequest(tenantInfo, customerID, billingqueue.BillTypeCreditMemo)
	req.InvoiceDate = 1_700_000_000
	memo, err := svc.PreviewMemo(t.Context(), req)

	require.NoError(t, err)
	assert.Empty(t, memo.Number, "a number is minted only when the memo is made")
	assert.Equal(t, invoice.StatusDraft, memo.Status)
	assert.Equal(t, invoice.ScopeMemo, memo.Scope)
	assert.Equal(t, invoice.MemoKindManual, memo.MemoKind)
	assert.True(t, memo.TotalAmount.Equal(decimal.RequireFromString("-75.5")),
		"got %s", memo.TotalAmount)
	require.Len(t, memo.Lines, 2)
}
