package invoicecreationservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
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

func TestBuildInvoiceEntityUsesTenantFallbackAndSignsCreditMemoAmounts(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	item := &billingqueue.BillingQueueItem{
		ID:             pulid.MustNew("bqi_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ShipmentID:     pulid.MustNew("shp_"),
		BillType:       billingqueue.BillTypeCreditMemo,
		Number:         "CM-1001",
	}
	shp := &shipment.Shipment{
		ID:                  item.ShipmentID,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(100)),
		OtherChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(35)),
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(135)),
		ProNumber:           "PRO123",
		BOL:                 "BOL123",
		ActualDeliveryDate:  new(int64(1_700_000_500)),
		AdditionalCharges: []*shipment.AdditionalCharge{
			{
				Method: accessorialcharge.MethodPerUnit,
				Unit:   2,
				Amount: decimal.NewFromInt(20),
			},
			nil,
		},
	}
	cus := &customer.Customer{
		ID:           pulid.MustNew("cus_"),
		Name:         "Acme Logistics",
		Code:         "ACME",
		AddressLine1: "100 Main",
		City:         "Nashville",
		PostalCode:   "37201",
		BillingProfile: &customer.CustomerBillingProfile{
			BillingCurrency: "CAD",
		},
	}
	control := &billingcontrol.BillingControl{
		DefaultPaymentTerm: billingcontrol.PaymentTermNet45,
	}

	svc := &Service{
		l:   zap.NewNop(),
		rec: invoiceaction.NewRecorder(invoiceaction.RecorderParams{Logger: zap.NewNop()}),
	}

	entity := buildEntity(t, svc, &buildInvoiceParams{
		Anchor:   item,
		Scope:    invoice.ScopeShipment,
		Customer: cus,
		Control:  control,
		Legs:     []*shipment.Shipment{shp},
	})

	require.NotNil(t, entity)
	assert.Equal(t, item.Number, entity.Number)
	assert.Equal(t, "Net45", string(entity.PaymentTerm))
	assert.Equal(t, "CAD", entity.CurrencyCode)
	assert.True(t, decimal.NewFromInt(-100).Equal(entity.SubtotalAmount), "entity.SubtotalAmount")
	assert.Equal(t, int64(-10000), entity.SubtotalAmountMinor)
	assert.True(t, decimal.NewFromInt(-40).Equal(entity.OtherAmount), "entity.OtherAmount")
	assert.Equal(t, int64(-4000), entity.OtherAmountMinor)
	assert.True(t, decimal.NewFromInt(-140).Equal(entity.TotalAmount), "entity.TotalAmount")
	assert.Equal(t, int64(-14000), entity.TotalAmountMinor)
	assert.Len(t, entity.Lines, 2)
	assert.True(t, decimal.NewFromInt(-100).Equal(entity.Lines[0].Amount), "entity.Lines[0].Amount")
	assert.Equal(t, int64(-10000), entity.Lines[0].AmountMinor)
	assert.True(t, decimal.NewFromInt(-40).Equal(entity.Lines[1].Amount), "entity.Lines[1].Amount")
	assert.Equal(t, int64(-4000), entity.Lines[1].AmountMinor)
	assert.True(t, entity.Lines[1].UnitPrice.Equal(decimal.NewFromInt(-20)))
	assert.Equal(t, shp.ActualDeliveryDate, entity.ServiceDate)
}

func TestBuildInvoiceEntityDerivesAccessorialTotalsFromLines(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	item := &billingqueue.BillingQueueItem{
		ID:             pulid.MustNew("bqi_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ShipmentID:     pulid.MustNew("shp_"),
		BillType:       billingqueue.BillTypeInvoice,
		Number:         "INV-1001",
	}
	shp := &shipment.Shipment{
		ID:                  item.ShipmentID,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(2_800)),
		OtherChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(150)),
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(2_950)),
		AdditionalCharges: []*shipment.AdditionalCharge{
			{
				Method: accessorialcharge.MethodPerUnit,
				Amount: decimal.RequireFromString("37.50"),
				Unit:   2,
				AccessorialCharge: &accessorialcharge.AccessorialCharge{
					Description: "Detention",
				},
			},
		},
	}
	cus := &customer.Customer{
		ID:   pulid.MustNew("cus_"),
		Name: "Acme Logistics",
	}

	entity := buildEntity(t, (&Service{
		l:   zap.NewNop(),
		rec: invoiceaction.NewRecorder(invoiceaction.RecorderParams{Logger: zap.NewNop()}),
	}), &buildInvoiceParams{
		Anchor:   item,
		Scope:    invoice.ScopeShipment,
		Customer: cus,
		Control: &billingcontrol.BillingControl{
			DefaultPaymentTerm: billingcontrol.PaymentTermNet30,
		},
		Legs: []*shipment.Shipment{shp},
	})

	require.NotNil(t, entity)
	require.Len(t, entity.Lines, 2)
	assert.True(t, decimal.NewFromInt(2_800).Equal(entity.SubtotalAmount), "entity.SubtotalAmount")
	assert.True(t, decimal.NewFromInt(75).Equal(entity.OtherAmount))
	assert.True(t, decimal.NewFromInt(2_875).Equal(entity.TotalAmount))
	assert.Equal(t, int64(7_500), entity.OtherAmountMinor)
	assert.Equal(t, int64(287_500), entity.TotalAmountMinor)
	assert.Equal(t, "Detention", entity.Lines[1].Description)
	assert.True(t, decimal.NewFromInt(2).Equal(entity.Lines[1].Quantity))
	assert.True(t, decimal.RequireFromString("37.50").Equal(entity.Lines[1].UnitPrice))
	assert.True(t, decimal.NewFromInt(75).Equal(entity.Lines[1].Amount))
	assert.Equal(t, int64(7_500), entity.Lines[1].AmountMinor)
}

func TestCreateFromApprovedBillingQueueItemReusesExpandedQueueShipment(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	billingQueueItemID := pulid.MustNew("bqi_")
	shipmentID := pulid.MustNew("shp_")
	customerID := pulid.MustNew("cus_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
	queueShipment := &shipment.Shipment{
		ID:                  shipmentID,
		OrganizationID:      orgID,
		BusinessUnitID:      buID,
		CustomerID:          customerID,
		ProNumber:           "PRO123",
		BOL:                 "BOL123",
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(100)),
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(100)),
	}
	queueItem := &billingqueue.BillingQueueItem{
		ID:             billingQueueItemID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ShipmentID:     shipmentID,
		Status:         billingqueue.StatusApproved,
		BillType:       billingqueue.BillTypeInvoice,
		Number:         "INV-1001",
		Shipment:       queueShipment,
	}

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByBillingQueueItemID(mock.Anything, invoiceports.GetInvoiceByBillingQueueItemIDRequest{
			BillingQueueItemID: billingQueueItemID,
			TenantInfo:         tenantInfo,
		}).
		Return(nil, errortypes.NewNotFoundError("invoice not found")).
		Once()
	repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(entity *invoice.Invoice) bool {
			return entity != nil &&
				entity.BillingQueueItemID == billingQueueItemID &&
				entity.ShipmentID == shipmentID &&
				entity.CustomerID == customerID &&
				entity.ShipmentProNumber == "PRO123" &&
				entity.TotalAmount.Equal(decimal.NewFromInt(100))
		})).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			entity.ID = pulid.MustNew("inv_")
			return entity, nil
		}).
		Once()

	billingQueueRepo := billingqueueportsmock.NewMockBillingQueueRepository(t)
	billingQueueRepo.EXPECT().
		GetByID(mock.Anything, &billingqueueports.GetBillingQueueItemByIDRequest{
			ItemID:                billingQueueItemID,
			TenantInfo:            tenantInfo,
			ExpandShipmentDetails: true,
		}).
		Return(queueItem, nil).
		Once()

	customerRepo := customerportsmock.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByID(mock.Anything, customerports.GetCustomerByIDRequest{
			ID:         customerID,
			TenantInfo: tenantInfo,
			CustomerFilterOptions: customerports.CustomerFilterOptions{
				IncludeBillingProfile: true,
				IncludeState:          true,
			},
		}).
		Return(&customer.Customer{
			ID:           customerID,
			Name:         "Acme Logistics",
			AddressLine1: "100 Main",
			City:         "Nashville",
			PostalCode:   "37201",
			BillingProfile: &customer.CustomerBillingProfile{
				BillingCurrency: "USD",
			},
		}, nil).
		Once()

	billingRepo := tenantportsmock.NewMockBillingControlRepository(t)
	billingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&billingcontrol.BillingControl{DefaultPaymentTerm: billingcontrol.PaymentTermNet30}, nil).
		Twice()
	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)

	svc := &Service{
		l:                zap.NewNop(),
		repo:             repo,
		billingQueueRepo: billingQueueRepo,
		shipmentRepo:     shipmentRepo,
		customerRepo:     customerRepo,
		billingRepo:      billingRepo,
		validator: invoicevalidation.NewValidator(
			invoicevalidation.ValidatorParams{Logger: zap.NewNop()},
		),
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{
				Logger:       zap.NewNop(),
				AuditService: &mocks.NoopAuditService{},
				Realtime:     &mocks.NoopRealtimeService{},
			},
		),
	}

	result, err := svc.CreateFromApprovedBillingQueueItem(
		t.Context(),
		&invoiceports.CreateInvoiceFromBillingQueueRequest{
			BillingQueueItemID: billingQueueItemID,
			TenantInfo:         tenantInfo,
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Invoice)
	assert.Equal(t, shipmentID, result.Invoice.ShipmentID)
}

func TestShouldAutoPost(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")

	t.Run(
		"organization auto posting requires both org automation and customer opt-in",
		func(t *testing.T) {
			t.Parallel()

			billingRepo := tenantportsmock.NewMockBillingControlRepository(t)
			billingRepo.EXPECT().
				GetByOrgID(t.Context(), orgID).
				Return(&billingcontrol.BillingControl{
					InvoiceDraftCreationMode: billingcontrol.InvoiceDraftCreationModeAutomaticWhenTransferred,
					InvoicePostingMode:       billingcontrol.InvoicePostingModeAutomaticWhenNoBlockingExceptions,
				}, nil)

			svc := &Service{
				l:           zap.NewNop(),
				billingRepo: billingRepo,
				rec: invoiceaction.NewRecorder(
					invoiceaction.RecorderParams{Logger: zap.NewNop()},
				),
			}

			autoPost := svc.shouldAutoPost(t.Context(), orgID, &customer.Customer{
				BillingProfile: &customer.CustomerBillingProfile{AutoBill: true},
			})

			assert.True(t, autoPost)
		},
	)

	t.Run("customer auto bill cannot loosen organization manual review policy", func(t *testing.T) {
		t.Parallel()

		billingRepo := tenantportsmock.NewMockBillingControlRepository(t)
		billingRepo.EXPECT().GetByOrgID(t.Context(), orgID).Return(&billingcontrol.BillingControl{
			InvoicePostingMode: billingcontrol.InvoicePostingModeManualReviewRequired,
		}, nil)

		svc := &Service{
			l:           zap.NewNop(),
			billingRepo: billingRepo,
			rec: invoiceaction.NewRecorder(
				invoiceaction.RecorderParams{Logger: zap.NewNop()},
			),
		}

		autoPost := svc.shouldAutoPost(t.Context(), orgID, &customer.Customer{
			BillingProfile: &customer.CustomerBillingProfile{AutoBill: true},
		})

		assert.False(t, autoPost)
	})

	t.Run("customer fallback still works on tenant not found", func(t *testing.T) {
		t.Parallel()

		billingRepo := tenantportsmock.NewMockBillingControlRepository(t)
		billingRepo.EXPECT().GetByOrgID(t.Context(), orgID).
			Return(nil, errortypes.NewNotFoundError("billing control not found"))

		svc := &Service{
			l:           zap.NewNop(),
			billingRepo: billingRepo,
			rec: invoiceaction.NewRecorder(
				invoiceaction.RecorderParams{Logger: zap.NewNop()},
			),
		}

		autoPost := svc.shouldAutoPost(t.Context(), orgID, &customer.Customer{
			BillingProfile: &customer.CustomerBillingProfile{AutoBill: true},
		})

		assert.True(t, autoPost)
	})
}
