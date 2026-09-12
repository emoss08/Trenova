package invoiceservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
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

type approvalHarness struct {
	svc         *Service
	tenantInfo  pagination.TenantInfo
	itemID      pulid.ID
	shipmentID  pulid.ID
	customerID  pulid.ID
	orgID, buID pulid.ID
	userID      pulid.ID
}

// newApprovalHarness wires the approve-to-invoice path with a customer whose
// billing profile the caller shapes.
//
// expectCreate is false for the statement case, and that is the assertion that
// matters most: the invoice repository is never asked to create anything, so a
// regression that bills a statement customer on approval fails on the mock
// rather than on a value comparison after the fact.
func newApprovalHarness(
	t *testing.T,
	profile *customer.CustomerBillingProfile,
	expectCreate bool,
) *approvalHarness {
	t.Helper()

	h := &approvalHarness{
		orgID:      pulid.MustNew("org_"),
		buID:       pulid.MustNew("bu_"),
		userID:     pulid.MustNew("usr_"),
		itemID:     pulid.MustNew("bqi_"),
		shipmentID: pulid.MustNew("shp_"),
		customerID: pulid.MustNew("cus_"),
	}
	h.tenantInfo = pagination.TenantInfo{OrgID: h.orgID, BuID: h.buID, UserID: h.userID}

	queueShipment := &shipment.Shipment{
		ID:                  h.shipmentID,
		OrganizationID:      h.orgID,
		BusinessUnitID:      h.buID,
		CustomerID:          h.customerID,
		ProNumber:           "PRO123",
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(100)),
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(100)),
	}
	queueItem := &billingqueue.BillingQueueItem{
		ID:             h.itemID,
		OrganizationID: h.orgID,
		BusinessUnitID: h.buID,
		ShipmentID:     h.shipmentID,
		Status:         billingqueue.StatusApproved,
		BillType:       billingqueue.BillTypeInvoice,
		Number:         "INV-1001",
		Shipment:       queueShipment,
	}

	repo := mocks.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByBillingQueueItemID(mock.Anything, repositories.GetInvoiceByBillingQueueItemIDRequest{
			BillingQueueItemID: h.itemID,
			TenantInfo:         h.tenantInfo,
		}).
		Return(nil, errortypes.NewNotFoundError("invoice not found")).
		Once()
	if expectCreate {
		repo.EXPECT().
			Create(mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
				entity.ID = pulid.MustNew("inv_")
				return entity, nil
			}).
			Once()
	}

	billingQueueRepo := mocks.NewMockBillingQueueRepository(t)
	billingQueueRepo.EXPECT().
		GetByID(mock.Anything, &repositories.GetBillingQueueItemByIDRequest{
			ItemID:                h.itemID,
			TenantInfo:            h.tenantInfo,
			ExpandShipmentDetails: true,
		}).
		Return(queueItem, nil).
		Once()

	customerRepo := mocks.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&customer.Customer{
			ID:             h.customerID,
			Name:           "GlobalTrade Imports",
			AddressLine1:   "100 Main",
			City:           "Nashville",
			PostalCode:     "37201",
			BillingProfile: profile,
		}, nil).
		Once()

	billingRepo := mocks.NewMockBillingControlRepository(t)
	billingRepo.EXPECT().
		GetByOrgID(mock.Anything, h.orgID).
		Return(&tenant.BillingControl{DefaultPaymentTerm: tenant.PaymentTermNet30}, nil).
		Maybe()

	h.svc = &Service{
		l:                zap.NewNop(),
		repo:             repo,
		billingQueueRepo: billingQueueRepo,
		shipmentRepo:     mocks.NewMockShipmentRepository(t),
		customerRepo:     customerRepo,
		billingRepo:      billingRepo,
		validator: &Validator{
			validator: validationframework.NewTenantedValidatorBuilder[*invoice.Invoice]().Build(),
		},
		auditService: &mocks.NoopAuditService{},
		realtime:     &mocks.NoopRealtimeService{},
	}

	return h
}

func (h *approvalHarness) approve(
	t *testing.T,
	req *servicesports.CreateInvoiceFromBillingQueueRequest,
) (*servicesports.CreateInvoiceFromBillingQueueResult, error) {
	t.Helper()
	req.BillingQueueItemID = h.itemID
	req.TenantInfo = h.tenantInfo

	return h.svc.CreateFromApprovedBillingQueueItem(
		t.Context(),
		req,
		testutil.NewSessionActor(h.userID, h.orgID, h.buID),
	)
}

func statementProfile() *customer.CustomerBillingProfile {
	return &customer.CustomerBillingProfile{
		BillingCurrency:       "USD",
		InvoiceDelivery:       customer.InvoiceDeliveryConsolidated,
		BillingCycle:          customer.BillingCycleMonthly,
		BillingCycleAnchorDay: 1,
	}
}

func perShipmentProfile() *customer.CustomerBillingProfile {
	return &customer.CustomerBillingProfile{
		BillingCurrency: "USD",
		InvoiceDelivery: customer.InvoiceDeliveryPerShipment,
		BillingCycle:    customer.BillingCycleImmediate,
	}
}

// Approving a statement customer's freight is how it reaches their statement.
// Refusing the approval — which the cadence guard did, because approval and
// invoicing were the same operation — left the item unapproved, and unapproved
// is the one state ListConsolidationCandidates excludes. The freight could never
// appear on the statement it belonged to.
func TestApprovingStatementFreightDefersInsteadOfInvoicing(t *testing.T) {
	t.Parallel()

	h := newApprovalHarness(t, statementProfile(), false)

	result, err := h.approve(t, &servicesports.CreateInvoiceFromBillingQueueRequest{
		DeferToStatement: true,
	})

	require.NoError(t, err, "approving onto a statement is the schedule working, not an error")
	require.NotNil(t, result)
	assert.True(t, result.DeferredToStatement)
	assert.Nil(t, result.Invoice, "the statement bills this later; approval must not")
}

func TestApprovingOrdinaryFreightStillInvoices(t *testing.T) {
	t.Parallel()

	h := newApprovalHarness(t, perShipmentProfile(), true)

	result, err := h.approve(t, &servicesports.CreateInvoiceFromBillingQueueRequest{
		DeferToStatement: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.DeferredToStatement)
	require.NotNil(t, result.Invoice, "a per-shipment customer is billed on approval as before")
}

// Deferral is asked for by the approve action alone. An explicit "invoice this
// shipment now" still has to justify taking the freight off the statement, or
// the guard would be unreachable.
func TestExplicitInvoicingStillDemandsAReason(t *testing.T) {
	t.Parallel()

	h := newApprovalHarness(t, statementProfile(), false)

	_, err := h.approve(t, &servicesports.CreateInvoiceFromBillingQueueRequest{})

	require.Error(t, err)
	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "offCycleReason", validationErr.Field)
}

func TestExplicitInvoicingProceedsWithAReason(t *testing.T) {
	t.Parallel()

	h := newApprovalHarness(t, statementProfile(), true)

	result, err := h.approve(t, &servicesports.CreateInvoiceFromBillingQueueRequest{
		OffCycleReason: "Customer is closing their books early",
	})

	require.NoError(t, err)
	require.NotNil(t, result.Invoice)
	assert.Equal(
		t,
		"Customer is closing their books early",
		result.Invoice.OffCycleReason,
		"the reason is stamped on the invoice it justifies",
	)
}
