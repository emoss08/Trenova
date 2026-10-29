package invoicecreationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

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

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().GetByBillingQueueItemID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("invoice not found")).Once()
	queue := billingqueueportsmock.NewMockBillingQueueRepository(t)
	queue.EXPECT().GetByID(mock.Anything, mock.MatchedBy(
		func(req *billingqueueports.GetBillingQueueItemByIDRequest) bool {
			return req.ItemID == item.ID && req.ExpandShipmentDetails
		},
	)).Return(item, nil).Once()
	customers := customerportsmock.NewMockCustomerRepository(t)
	customers.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&customer.Customer{
		ID:             customerID,
		Name:           "Acme Logistics",
		BillingProfile: &customer.CustomerBillingProfile{AutoBill: true, BillingCurrency: "USD"},
	}, nil).Once()
	billing := tenantportsmock.NewMockBillingControlRepository(t)
	billing.EXPECT().GetByOrgID(mock.Anything, orgID).
		Return(nil, errortypes.NewNotFoundError("no billing control")).Twice()

	svc := &Service{
		l:                zap.NewNop(),
		repo:             repo,
		billingQueueRepo: queue,
		shipmentRepo:     shipmentportsmock.NewMockShipmentRepository(t),
		customerRepo:     customers,
		billingRepo:      billing,
		validator: invoicevalidation.NewValidator(
			invoicevalidation.ValidatorParams{Logger: zap.NewNop()},
		),
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{Logger: zap.NewNop()},
		),
	}

	result, err := svc.PreviewApprovalInvoice(
		t.Context(),
		&invoiceports.CreateInvoiceFromBillingQueueRequest{
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
