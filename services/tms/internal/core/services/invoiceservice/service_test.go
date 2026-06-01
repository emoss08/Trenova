package invoiceservice

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/chai2010/webp"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	portstorage "github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/temporaljobs/billingjobs"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	storagetest "github.com/emoss08/trenova/shared/testutil/storage"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type fakeInvoiceDocumentService struct {
	currentDocument *document.Document
}

func (f *fakeInvoiceDocumentService) Get(
	_ context.Context,
	req repositories.GetDocumentByIDRequest,
) (*document.Document, error) {
	if f.currentDocument != nil {
		return f.currentDocument, nil
	}
	return &document.Document{ID: req.ID, LineageID: req.ID}, nil
}

func (f *fakeInvoiceDocumentService) GetDownloadContent(
	context.Context,
	repositories.GetDocumentByIDRequest,
) (*servicesports.DocumentContent, error) {
	return nil, nil
}

type fakeWorkflowRun struct {
	id    string
	runID string
}

func (f fakeWorkflowRun) GetID() string {
	return f.id
}

func (f fakeWorkflowRun) GetRunID() string {
	return f.runID
}

func (f fakeWorkflowRun) Get(context.Context, interface{}) error {
	return nil
}

func (f fakeWorkflowRun) GetWithOptions(
	context.Context,
	interface{},
	client.WorkflowRunGetOptions,
) error {
	return nil
}

func TestResolvePaymentTerm(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		customer *customer.Customer
		control  *tenant.BillingControl
		expected string
	}{
		{
			name: "customer override wins",
			customer: &customer.Customer{
				BillingProfile: &customer.CustomerBillingProfile{
					PaymentTerm: customer.PaymentTermNet15,
				},
			},
			control: &tenant.BillingControl{
				DefaultPaymentTerm: tenant.PaymentTermNet60,
			},
			expected: "Net15",
		},
		{
			name: "tenant fallback used when customer term missing",
			customer: &customer.Customer{
				BillingProfile: &customer.CustomerBillingProfile{},
			},
			control: &tenant.BillingControl{
				DefaultPaymentTerm: tenant.PaymentTermNet45,
			},
			expected: "Net45",
		},
		{
			name:     "empty when neither source has a term",
			customer: &customer.Customer{},
			control:  nil,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, string(resolvePaymentTerm(tc.customer, tc.control)))
		})
	}
}

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
		ActualDeliveryDate:  int64Ptr(1_700_000_500),
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
	control := &tenant.BillingControl{
		DefaultPaymentTerm: tenant.PaymentTermNet45,
	}

	svc := &Service{l: zap.NewNop()}

	entity := svc.buildInvoiceEntity(item, shp, cus, control)

	require.NotNil(t, entity)
	assert.Equal(t, item.Number, entity.Number)
	assert.Equal(t, "Net45", string(entity.PaymentTerm))
	assert.Equal(t, "CAD", entity.CurrencyCode)
	assert.Equal(t, decimal.NewFromInt(-100), entity.SubtotalAmount)
	assert.Equal(t, int64(-10000), entity.SubtotalAmountMinor)
	assert.Equal(t, decimal.NewFromInt(-40), entity.OtherAmount)
	assert.Equal(t, int64(-4000), entity.OtherAmountMinor)
	assert.Equal(t, decimal.NewFromInt(-140), entity.TotalAmount)
	assert.Equal(t, int64(-14000), entity.TotalAmountMinor)
	assert.Len(t, entity.Lines, 2)
	assert.Equal(t, decimal.NewFromInt(-100), entity.Lines[0].Amount)
	assert.Equal(t, int64(-10000), entity.Lines[0].AmountMinor)
	assert.Equal(t, decimal.NewFromInt(-40), entity.Lines[1].Amount)
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

	entity := (&Service{l: zap.NewNop()}).buildInvoiceEntity(
		item,
		shp,
		cus,
		&tenant.BillingControl{DefaultPaymentTerm: tenant.PaymentTermNet30},
	)

	require.NotNil(t, entity)
	require.Len(t, entity.Lines, 2)
	assert.Equal(t, decimal.NewFromInt(2_800), entity.SubtotalAmount)
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

	repo := mocks.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByBillingQueueItemID(mock.Anything, repositories.GetInvoiceByBillingQueueItemIDRequest{
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

	billingQueueRepo := mocks.NewMockBillingQueueRepository(t)
	billingQueueRepo.EXPECT().
		GetByID(mock.Anything, &repositories.GetBillingQueueItemByIDRequest{
			ItemID:                billingQueueItemID,
			TenantInfo:            tenantInfo,
			ExpandShipmentDetails: true,
		}).
		Return(queueItem, nil).
		Once()

	customerRepo := mocks.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetCustomerByIDRequest{
			ID:         customerID,
			TenantInfo: tenantInfo,
			CustomerFilterOptions: repositories.CustomerFilterOptions{
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

	billingRepo := mocks.NewMockBillingControlRepository(t)
	billingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&tenant.BillingControl{DefaultPaymentTerm: tenant.PaymentTermNet30}, nil).
		Twice()
	shipmentRepo := mocks.NewMockShipmentRepository(t)

	svc := &Service{
		l:                zap.NewNop(),
		repo:             repo,
		billingQueueRepo: billingQueueRepo,
		shipmentRepo:     shipmentRepo,
		customerRepo:     customerRepo,
		billingRepo:      billingRepo,
		validator: &Validator{
			validator: validationframework.NewTenantedValidatorBuilder[*invoice.Invoice]().Build(),
		},
		auditService: &mocks.NoopAuditService{},
		realtime:     &mocks.NoopRealtimeService{},
	}

	result, err := svc.CreateFromApprovedBillingQueueItem(
		t.Context(),
		&servicesports.CreateInvoiceFromBillingQueueRequest{
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

func TestInvoiceDeliveryTemplates(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{
		Number:                 "INV-1001",
		InvoiceDate:            1_700_000_000,
		DueDate:                int64Ptr(1_700_086_400),
		PaymentTerm:            invoice.PaymentTermNet30,
		CurrencyCode:           "USD",
		TotalAmount:            decimal.NewFromInt(1250),
		BillToName:             "Acme Logistics",
		BillToCode:             "ACME",
		ShipmentProNumber:      "PRO123",
		ShipmentBOL:            "BOL123",
		RemittanceInstructions: "ACH preferred",
	}
	context := invoiceTemplateContext(entity, &invoiceDeliveryProfile{
		Organization: &tenant.Organization{Name: "Trenova Freight"},
		Shipment: &shipment.Shipment{
			ProNumber: "PRO123",
			BOL:       "BOL123",
		},
	})

	rendered := renderInvoiceTemplate(
		"Invoice #{number} for {customer} from {company}; {{invoice.number}} {{customer.name}} {{missing.value}}",
		context,
	)

	require.Equal(
		t,
		"Invoice #INV-1001 for Acme Logistics from Trenova Freight; INV-1001 Acme Logistics {{missing.value}}",
		rendered.Value,
	)
	require.Equal(t, []string{"missing.value"}, rendered.Unknown)
}

func TestResolveSubjectRendersDraftSnapshotBeforeCustomerTemplate(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{
		Number:               "INV-1001",
		BillToName:           "Acme Logistics",
		EmailSubjectSnapshot: "Draft invoice #{number} for {customer}",
	}
	profile := &customer.CustomerEmailProfile{
		Subject: "Customer {{invoice.number}}",
	}

	result := resolveSubject(entity, profile, invoiceTemplateContext(entity, nil))

	require.Equal(t, "Draft invoice #INV-1001 for Acme Logistics", result.Value)
	require.Empty(t, result.Unknown)
}

func TestResolveBodyRendersDraftSnapshot(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{
		Number:            "INV-1001",
		BillToName:        "Acme Logistics",
		EmailBodySnapshot: "Please review {number}, {customer}.",
	}

	result := resolveBody(entity, nil, invoiceTemplateContext(entity, nil))

	require.Equal(t, "Please review INV-1001, Acme Logistics.", result.Value)
	require.Empty(t, result.Unknown)
}

func TestInvoicePDFAttachmentNameSanitizesAndForcesPDF(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{Number: "INV/1001"}

	name := invoicePDFAttachmentName("../Invoice: {{bad}}.xlsx", entity)

	require.Equal(t, "Invoice- {{bad}}.pdf", name)
}

func TestBuildInvoicePDFDataMapsBillToAndRemitSeparately(t *testing.T) {
	t.Parallel()

	state := &usstate.UsState{Abbreviation: "TN", CountryName: "USA"}
	entity := &invoice.Invoice{
		Number:             "INV-1001",
		BillToName:         "Snapshot Customer",
		BillToAddressLine1: "100 Snapshot Ave",
		BillToCity:         "Nashville",
		BillToPostalCode:   "37201",
		RemittanceInstructions: strings.Join([]string{
			"ACH preferred",
			"Account ending 4321",
		}, "\n"),
		CurrencyCode: "USD",
	}
	cus := &customer.Customer{
		Name:         "Fallback Customer",
		AddressLine1: "999 Customer Rd",
		AddressLine2: "Suite AP",
		City:         "Memphis",
		PostalCode:   "38103",
		State:        state,
	}
	org := &tenant.Organization{
		Name:         "Carrier Organization",
		AddressLine1: "500 Remit St",
		City:         "Knoxville",
		PostalCode:   "37902",
		State:        state,
	}

	data := buildInvoicePDFData(entity, &invoiceDeliveryProfile{
		Customer:     cus,
		Organization: org,
	})

	require.Equal(t, "Snapshot Customer", data.BillTo.Name)
	require.Contains(t, data.BillTo.Lines, "100 Snapshot Ave")
	require.Contains(t, data.BillTo.Lines, "Suite AP")
	require.NotContains(t, append([]string{data.BillTo.Name}, data.BillTo.Lines...), "Carrier Organization")
	require.Equal(t, "Carrier Organization", data.RemitTo.Name)
	require.Contains(t, data.RemitTo.Lines, "500 Remit St")
	require.Contains(t, data.RemitTo.Lines, "ACH preferred")
	require.NotContains(t, append([]string{data.RemitTo.Name}, data.RemitTo.Lines...), "Fallback Customer")
}

func TestBuildInvoicePDFDataMapsShipmentStopsToShipperAndConsignee(t *testing.T) {
	t.Parallel()

	state := &usstate.UsState{Abbreviation: "GA", CountryName: "USA"}
	pickupDate := int64(1_700_000_000)
	deliveryDate := int64(1_700_086_400)
	shp := &shipment.Shipment{
		ActualShipDate:     &pickupDate,
		ActualDeliveryDate: &deliveryDate,
		Moves: []*shipment.ShipmentMove{
			{
				Stops: []*shipment.Stop{
					{
						Type:     shipment.StopTypeDelivery,
						Sequence: 2,
						Location: &location.Location{
							Name:         "Consignee DC",
							AddressLine1: "200 Delivery Ln",
							City:         "Atlanta",
							PostalCode:   "30301",
							State:        state,
						},
					},
					{
						Type:     shipment.StopTypeDelivery,
						Sequence: 3,
						Location: &location.Location{
							Name:         "Final Delivery",
							AddressLine1: "300 Final Ln",
							City:         "Macon",
							PostalCode:   "31201",
							State:        state,
						},
					},
					{
						Type:     shipment.StopTypePickup,
						Sequence: 1,
						Location: &location.Location{
							Name:         "Shipper Plant",
							AddressLine1: "100 Pickup Rd",
							City:         "Savannah",
							PostalCode:   "31401",
							State:        state,
						},
					},
				},
			},
		},
	}

	data := buildInvoicePDFData(&invoice.Invoice{CurrencyCode: "USD"}, &invoiceDeliveryProfile{Shipment: shp})

	require.Equal(t, "Shipper Plant", data.Shipper.Name)
	require.Contains(t, data.Shipper.Lines, "100 Pickup Rd")
	require.Equal(t, "Consignee DC", data.Consignee.Name)
	require.Contains(t, data.Consignee.Lines, "200 Delivery Ln")
	require.NotEqual(t, data.Shipper.Name, data.Consignee.Name)
	require.Equal(t, []invoicePDFKeyValue{
		{Label: "Pickup Date", Value: unixDatePtr(&pickupDate)},
	}, data.Shipper.Details)
	require.Equal(t, []invoicePDFKeyValue{
		{Label: "Delivery Date", Value: unixDatePtr(&deliveryDate)},
	}, data.Consignee.Details)
}

func TestShipmentStopSelectionHelpers(t *testing.T) {
	t.Parallel()

	shp := &shipment.Shipment{
		Moves: []*shipment.ShipmentMove{
			{
				Stops: []*shipment.Stop{
					{
						Type:     shipment.StopTypeDelivery,
						Sequence: 4,
						Location: &location.Location{
							Name:       "Final Delivery",
							City:       "Macon",
							PostalCode: "31201",
						},
					},
					{
						Type:     shipment.StopTypePickup,
						Sequence: 2,
						Location: &location.Location{
							Name:       "First Pickup",
							City:       "Savannah",
							PostalCode: "31401",
						},
					},
					{
						Type:     shipment.StopTypeDelivery,
						Sequence: 3,
						Location: &location.Location{
							Name:       "First Delivery",
							City:       "Atlanta",
							PostalCode: "30301",
						},
					},
					{
						Type:     shipment.StopTypePickup,
						Sequence: 5,
						Location: &location.Location{Name: "Later Pickup"},
					},
				},
			},
		},
	}

	require.Equal(t, int64(2), firstPickupStop(shp).Sequence)
	require.Equal(t, int64(3), firstDeliveryStop(shp).Sequence)
	require.Equal(t, int64(4), finalDeliveryStop(shp).Sequence)
	require.Equal(t, "First Pickup Savannah 31401", shipmentOrigin(shp))
	require.Equal(t, "Final Delivery Macon 31201", shipmentDestination(shp))
}

func TestBuildInvoicePDFDataMapsShipmentCommodities(t *testing.T) {
	t.Parallel()

	shp := &shipment.Shipment{
		Commodities: []*shipment.ShipmentCommodity{
			{
				Pieces: 20,
				Weight: 40000,
				Commodity: &commodity.Commodity{
					Name:         "Industrial Parts",
					Description:  "SKU 12345",
					FreightClass: commodity.FreightClass125,
				},
			},
			nil,
		},
	}

	data := buildInvoicePDFData(&invoice.Invoice{CurrencyCode: "USD"}, &invoiceDeliveryProfile{
		Shipment: shp,
	})

	require.Equal(t, []invoicePDFCommodityRow{
		{
			Quantity:         "20",
			DescriptionLines: []string{"Industrial Parts", "SKU 12345"},
			Weight:           "40,000",
			Class:            "125",
			PiecesValue:      20,
			WeightValue:      40000,
		},
	}, data.CommodityRows)
}

func TestBuildInvoicePDFDataOmitsMissingOptionalRows(t *testing.T) {
	t.Parallel()

	data := buildInvoicePDFData(&invoice.Invoice{
		Number:                 "INV-1001",
		CurrencyCode:           "USD",
		RemittanceInstructions: "\nACH only\n\n",
	}, &invoiceDeliveryProfile{
		Organization: &tenant.Organization{Name: "Carrier Organization"},
	})

	require.Equal(t, []string{"ACH only"}, data.RemitTo.Lines)
	require.Empty(t, data.Terms)
	require.NotContains(t, data.RemitTo.Lines, "")
	require.NotContains(t, data.BillTo.Lines, "")
	require.Empty(t, data.CommodityRows)
	require.Empty(t, data.InvoiceTerms)
	require.Empty(t, data.InvoiceFooter)
}

func TestBuildInvoicePDFDataUsesBillingControlTermsAndFooter(t *testing.T) {
	t.Parallel()

	data := buildInvoicePDFData(&invoice.Invoice{CurrencyCode: "USD"}, &invoiceDeliveryProfile{
		BillingControl: &tenant.BillingControl{
			DefaultInvoiceTerms:  "\nFirst invoice term.\n\nSecond invoice term.\n",
			DefaultInvoiceFooter: "  Thank you for your business.  ",
		},
	})

	require.Equal(t, []string{"First invoice term.", "Second invoice term."}, data.InvoiceTerms)
	require.Equal(t, "Thank you for your business.", data.InvoiceFooter)
}

func TestBuildInvoicePDFDataUsesBillingControlInvoiceDisplayOptions(t *testing.T) {
	t.Parallel()

	dueDate := int64(1_700_086_400)
	entity := &invoice.Invoice{
		CurrencyCode:   "USD",
		PaymentTerm:    invoice.PaymentTermNet30,
		DueDate:        &dueDate,
		TotalAmount:    decimal.NewFromInt(1250),
		AppliedAmount:  decimal.NewFromInt(250),
		SubtotalAmount: decimal.NewFromInt(1250),
	}

	defaulted := buildInvoicePDFData(entity, nil)
	require.Equal(t, "2023-11-15", defaulted.DueDate)
	require.Equal(t, "USD 1000.00", defaulted.BalanceDue)
	require.Contains(t, defaulted.Terms, "Due Date: 2023-11-15")

	shown := buildInvoicePDFData(entity, &invoiceDeliveryProfile{
		BillingControl: &tenant.BillingControl{
			ShowDueDateOnInvoice:    true,
			ShowBalanceDueOnInvoice: true,
		},
	})
	require.Equal(t, "2023-11-15", shown.DueDate)
	require.Equal(t, "USD 1000.00", shown.BalanceDue)
	require.Contains(t, shown.Terms, "Due Date: 2023-11-15")

	hidden := buildInvoicePDFData(entity, &invoiceDeliveryProfile{
		BillingControl: &tenant.BillingControl{},
	})
	require.Empty(t, hidden.DueDate)
	require.Empty(t, hidden.BalanceDue)
	require.Equal(t, []string{"Payment Terms: Net30"}, hidden.Terms)
}

func TestResolveDeliveryProfileIncludesBillingControlWhenRequested(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	control := &tenant.BillingControl{
		DefaultInvoiceTerms:  "Invoice terms",
		DefaultInvoiceFooter: "Invoice footer",
	}
	billingRepo := mocks.NewMockBillingControlRepository(t)
	billingRepo.EXPECT().GetByOrgID(t.Context(), orgID).Return(control, nil).Once()

	svc := &Service{billingRepo: billingRepo}

	profile, err := svc.resolveDeliveryProfile(t.Context(), resolveDeliveryProfileParams{
		Entity:                &invoice.Invoice{},
		TenantInfo:            pagination.TenantInfo{OrgID: orgID},
		IncludeBillingControl: true,
	})

	require.NoError(t, err)
	require.Same(t, control, profile.BillingControl)
}

func TestResolveDeliveryProfileSkipsBillingControlWhenNotRequested(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	billingRepo := mocks.NewMockBillingControlRepository(t)
	svc := &Service{billingRepo: billingRepo}

	profile, err := svc.resolveDeliveryProfile(t.Context(), resolveDeliveryProfileParams{
		Entity:     &invoice.Invoice{},
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	})

	require.NoError(t, err)
	require.Nil(t, profile.BillingControl)
}

func TestBuildInvoicePDFDataChargesMatchInvoiceTotals(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{
		CurrencyCode:   "USD",
		SubtotalAmount: decimal.NewFromInt(100),
		OtherAmount:    decimal.NewFromInt(25),
		TotalAmount:    decimal.NewFromInt(125),
		Lines: []*invoice.InoviceLine{
			{
				LineNumber:  1,
				Description: "Freight",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(100),
				Amount:      decimal.NewFromInt(100),
			},
			{
				LineNumber:  2,
				Description: "Detention",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(25),
				Amount:      decimal.NewFromInt(25),
			},
		},
	}

	data := buildInvoicePDFData(entity, nil)

	require.Len(t, data.ChargeRows, 2)
	require.Equal(t, "USD 100.00", data.Subtotal)
	require.Equal(t, "USD 25.00", data.Other)
	require.Equal(t, "USD 125.00", data.Total)
	require.Equal(t, "USD 100.00", data.ChargeRows[0].Amount)
	require.Equal(t, "USD 25.00", data.ChargeRows[1].Amount)
}

func TestBuildInvoicePDFDataEmbedsStoredPNGLogo(t *testing.T) {
	t.Parallel()

	logoBytes := testPNGLogo(t)
	storageClient := storagetest.NewMockStorageClient()
	storageClient.DownloadFunc = func(_ context.Context, key string) (*portstorage.DownloadResult, error) {
		require.Equal(t, "logos/org.png", key)
		return &portstorage.DownloadResult{
			Body:        io.NopCloser(bytes.NewReader(logoBytes)),
			ContentType: "image/png",
			Size:        int64(len(logoBytes)),
		}, nil
	}

	data := buildInvoicePDFDataWithLogo(
		t.Context(),
		&invoice.Invoice{CurrencyCode: "USD"},
		&invoiceDeliveryProfile{Organization: &tenant.Organization{Name: "Trenova", LogoURL: "logos/org.png"}},
		storageClient,
	)

	require.NotNil(t, data.Logo)
	require.Equal(t, "PNG", data.Logo.ImageType)
	require.Equal(t, logoBytes, data.Logo.Data)
}

func TestBuildInvoicePDFDataEmbedsStoredJPEGLogo(t *testing.T) {
	t.Parallel()

	logoBytes := testJPEGLogo(t)
	storageClient := storagetest.NewMockStorageClient()
	storageClient.DownloadFunc = func(context.Context, string) (*portstorage.DownloadResult, error) {
		return &portstorage.DownloadResult{
			Body:        io.NopCloser(bytes.NewReader(logoBytes)),
			ContentType: "image/jpeg",
			Size:        int64(len(logoBytes)),
		}, nil
	}

	data := buildInvoicePDFDataWithLogo(
		t.Context(),
		&invoice.Invoice{CurrencyCode: "USD"},
		&invoiceDeliveryProfile{Organization: &tenant.Organization{Name: "Trenova", LogoURL: "logos/org.jpg"}},
		storageClient,
	)

	require.NotNil(t, data.Logo)
	require.Equal(t, "JPG", data.Logo.ImageType)
	require.Equal(t, logoBytes, data.Logo.Data)
}

func TestBuildInvoicePDFDataConvertsStoredWebPLogoToPNG(t *testing.T) {
	t.Parallel()

	webpBytes := testWebPLogo(t)
	storageClient := storagetest.NewMockStorageClient()
	storageClient.DownloadFunc = func(context.Context, string) (*portstorage.DownloadResult, error) {
		return &portstorage.DownloadResult{
			Body:        io.NopCloser(bytes.NewReader(webpBytes)),
			ContentType: "image/webp",
			Size:        int64(len(webpBytes)),
		}, nil
	}

	data := buildInvoicePDFDataWithLogo(
		t.Context(),
		&invoice.Invoice{CurrencyCode: "USD"},
		&invoiceDeliveryProfile{Organization: &tenant.Organization{Name: "Trenova", LogoURL: "logos/org.webp"}},
		storageClient,
	)

	require.NotNil(t, data.Logo)
	require.Equal(t, "PNG", data.Logo.ImageType)
	require.True(t, bytes.HasPrefix(data.Logo.Data, []byte{0x89, 'P', 'N', 'G'}))
}

func TestFittedImageSizePreservesAspectRatioWithinBounds(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		sourceWidth    float64
		sourceHeight   float64
		maxWidth       float64
		maxHeight      float64
		expectedWidth  float64
		expectedHeight float64
	}{
		{
			name:           "landscape image caps by height",
			sourceWidth:    400,
			sourceHeight:   100,
			maxWidth:       78,
			maxHeight:      15,
			expectedWidth:  60,
			expectedHeight: 15,
		},
		{
			name:           "tall image caps by height",
			sourceWidth:    100,
			sourceHeight:   400,
			maxWidth:       78,
			maxHeight:      15,
			expectedWidth:  3.75,
			expectedHeight: 15,
		},
		{
			name:           "square image stays inside both caps",
			sourceWidth:    100,
			sourceHeight:   100,
			maxWidth:       78,
			maxHeight:      15,
			expectedWidth:  15,
			expectedHeight: 15,
		},
		{
			name:           "invalid source returns caps",
			sourceWidth:    0,
			sourceHeight:   100,
			maxWidth:       78,
			maxHeight:      15,
			expectedWidth:  78,
			expectedHeight: 15,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			width, height := fittedImageSize(
				tc.sourceWidth,
				tc.sourceHeight,
				tc.maxWidth,
				tc.maxHeight,
			)

			assert.InDelta(t, tc.expectedWidth, width, 0.001)
			assert.InDelta(t, tc.expectedHeight, height, 0.001)
			assert.LessOrEqual(t, width, tc.maxWidth)
			assert.LessOrEqual(t, height, tc.maxHeight)
		})
	}
}

func TestBuildInvoicePDFDataFallsBackWhenLogoUnsupported(t *testing.T) {
	t.Parallel()

	storageClient := storagetest.NewMockStorageClient()
	storageClient.DownloadFunc = func(context.Context, string) (*portstorage.DownloadResult, error) {
		return &portstorage.DownloadResult{
			Body:        io.NopCloser(strings.NewReader("not an image")),
			ContentType: "text/plain",
			Size:        int64(len("not an image")),
		}, nil
	}

	data := buildInvoicePDFDataWithLogo(
		t.Context(),
		&invoice.Invoice{CurrencyCode: "USD"},
		&invoiceDeliveryProfile{Organization: &tenant.Organization{Name: "Trenova", LogoURL: "logos/org.txt"}},
		storageClient,
	)

	require.Nil(t, data.Logo)
	require.Equal(t, "Trenova", data.Organization.Name)
}

func TestBuildInvoicePDFDataHeaderRowsOmitUnavailableMetadata(t *testing.T) {
	t.Parallel()

	data := buildInvoicePDFData(&invoice.Invoice{
		CurrencyCode:      "USD",
		PaymentTerm:       invoice.PaymentTermNet30,
		ShipmentProNumber: "PRO123",
	}, &invoiceDeliveryProfile{
		Organization: &tenant.Organization{
			DOTNumber: "1234567",
			ScacCode:  "TRNV",
		},
	})

	require.Equal(t, []invoicePDFKeyValue{
		{Label: "DOT", Value: "1234567"},
		{Label: "SCAC", Value: "TRNV"},
		{Label: "Payment Terms", Value: "Net30"},
		{Label: "PRO", Value: "PRO123"},
	}, data.HeaderRows)

	missing := buildInvoicePDFData(
		&invoice.Invoice{CurrencyCode: "USD"},
		&invoiceDeliveryProfile{Organization: &tenant.Organization{}},
	)
	require.Empty(t, missing.HeaderRows)
}

func TestInvoicePreviewForEntityReturnsPDFResult(t *testing.T) {
	t.Parallel()

	invoiceID := pulid.MustNew("inv_")
	entity := testInvoiceForPDF(invoiceID, pulid.MustNew("org_"), pulid.MustNew("bu_"))

	preview, err := invoicePreviewForEntity(t.Context(), entity, &invoiceDeliveryProfile{
		Organization: &tenant.Organization{Name: "Carrier Organization"},
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, preview)
	require.Equal(t, "application/pdf", preview.ContentType)
	require.Equal(t, "invoice-INV-1001.pdf", preview.FileName)
	require.NotEmpty(t, preview.Content)
	require.Equal(t, int64(len(preview.Content)), preview.SizeBytes)
}

func TestRenderPreviewLoadsBillingControl(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	invoiceID := pulid.MustNew("inv_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	entity := testInvoiceForPDF(invoiceID, orgID, buID)

	repo := mocks.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceByIDRequest{
			ID:         invoiceID,
			TenantInfo: tenantInfo,
		}).
		Return(entity, nil).
		Once()

	billingRepo := mocks.NewMockBillingControlRepository(t)
	billingRepo.EXPECT().
		GetByOrgID(mock.Anything, orgID).
		Return(&tenant.BillingControl{
			DefaultInvoiceTerms:  "Invoice terms",
			DefaultInvoiceFooter: "Invoice footer",
		}, nil).
		Once()

	svc := &Service{
		l:           zap.NewNop(),
		repo:        repo,
		billingRepo: billingRepo,
	}

	preview, err := svc.RenderPreview(t.Context(), &servicesports.InvoicePreviewRequest{
		InvoiceID:  invoiceID,
		TenantInfo: tenantInfo,
	})

	require.NoError(t, err)
	require.NotNil(t, preview)
	require.NotEmpty(t, preview.Content)
}

func TestInvoicePreviewWithFooterFitsSinglePage(t *testing.T) {
	t.Parallel()

	entity := testInvoiceForPDF(pulid.MustNew("inv_"), pulid.MustNew("org_"), pulid.MustNew("bu_"))
	entity.Lines = []*invoice.InoviceLine{
		{
			LineNumber:  1,
			Type:        invoice.InvoiceLineTypeFreight,
			Description: "Freight charge",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   decimal.NewFromInt(2800),
			Amount:      decimal.NewFromInt(2800),
		},
		{
			LineNumber:  2,
			Type:        invoice.InvoiceLineTypeAccessorial,
			Description: "Detention Fee",
			Quantity:    decimal.NewFromInt(2),
			UnitPrice:   decimal.NewFromInt(75),
			Amount:      decimal.NewFromInt(150),
		},
	}
	entity.SubtotalAmount = decimal.NewFromInt(2800)
	entity.OtherAmount = decimal.NewFromInt(150)
	entity.TotalAmount = decimal.NewFromInt(2950)

	preview, err := invoicePreviewForEntity(t.Context(), entity, testInvoicePDFDeliveryProfile(), nil)

	require.NoError(t, err)
	require.NotEmpty(t, preview.Content)
	require.Equal(t, 1, countPDFPages(preview.Content))
}

func TestPlanSendRendersTemplatesAndOrganizationAlias(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	invoiceID := pulid.MustNew("inv_")
	customerID := pulid.MustNew("cus_")
	documentID := pulid.MustNew("doc_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
	entity := &invoice.Invoice{
		ID:             invoiceID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CustomerID:     customerID,
		Number:         "INV-1001",
		CurrencyCode:   "USD",
		BillToName:     "Snapshot Customer",
		PDFDocumentID:  documentID,
		PDFDocument: &document.Document{
			ID:           documentID,
			OriginalName: "invoice-INV-1001.pdf",
			FileType:     "application/pdf",
			FileSize:     512,
		},
	}

	repo := mocks.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceByIDRequest{
			ID:         invoiceID,
			TenantInfo: tenantInfo,
		}).
		Return(entity, nil).
		Once()
	repo.EXPECT().
		ListAttachments(mock.Anything, repositories.ListInvoiceEmailAttemptsRequest{
			InvoiceID:  invoiceID,
			TenantInfo: tenantInfo,
		}).
		Return([]*invoice.Attachment{}, nil).
		Once()

	customerRepo := mocks.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetCustomerByIDRequest{
			ID:         customerID,
			TenantInfo: tenantInfo,
			CustomerFilterOptions: repositories.CustomerFilterOptions{
				IncludeEmailProfile: true,
			},
		}).
		Return(&customer.Customer{
			ID:   customerID,
			Name: "Acme Logistics",
			Code: "ACME",
			EmailProfile: &customer.CustomerEmailProfile{
				Subject:        "Invoice #{number} for {customer} from {company}",
				Comment:        "Invoice {{invoice.number}} for {{customer.name}} from {{organization.name}}",
				AttachmentName: "Invoice-{number}-{customer}-{company}.pdf",
				ToRecipients:   "ap@example.com",
			},
		}, nil).
		Once()

	organizationRepo := mocks.NewMockOrganizationRepository(t)
	organizationRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetOrganizationByIDRequest{TenantInfo: tenantInfo}).
		Return(&tenant.Organization{Name: "Trenova Freight"}, nil).
		Once()

	emailRepo := mocks.NewMockEmailRepository(t)
	emailRepo.EXPECT().
		GetAssignedProfile(mock.Anything, tenantInfo, email.PurposeBilling).
		Return(&email.Profile{SenderEmail: "billing@example.com"}, nil).
		Once()

	billingRepo := mocks.NewMockBillingControlRepository(t)
	svc := &Service{
		l:                zap.NewNop(),
		repo:             repo,
		customerRepo:     customerRepo,
		organizationRepo: organizationRepo,
		emailRepo:        emailRepo,
		billingRepo:      billingRepo,
	}

	plan, err := svc.PlanSend(t.Context(), &servicesports.InvoiceSendPlanRequest{
		InvoiceID:  invoiceID,
		TenantInfo: tenantInfo,
	})

	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Empty(t, plan.Errors)
	require.Empty(t, plan.Warnings)
	require.Len(t, plan.Parts, 1)
	require.Len(t, plan.Parts[0].Attachments, 1)
	assert.Equal(t, "Invoice #INV-1001 for Acme Logistics from Trenova Freight", plan.Subject)
	assert.Equal(t, "Invoice INV-1001 for Acme Logistics from Trenova Freight", plan.Body)
	assert.Equal(t, "Invoice-INV-1001-Acme Logistics-Trenova Freight.pdf", plan.Parts[0].Attachments[0].FileName)
}

func TestGeneratePDFStartsWorkflow(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	invoiceID := pulid.MustNew("inv_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	workflowStarter := mocks.NewMockWorkflowStarter(t)
	workflowStarter.EXPECT().Enabled().Return(true).Once()
	workflowStarter.EXPECT().
		StartWorkflow(
			mock.Anything,
			mock.MatchedBy(func(options client.StartWorkflowOptions) bool {
				return strings.HasPrefix(options.ID, "invoice-pdf-generate-"+invoiceID.String()) &&
					options.TaskQueue == temporaltype.TaskQueueBilling.String()
			}),
			billingjobs.GenerateInvoicePDFWorkflowName,
			mock.MatchedBy(func(args []any) bool {
				require.Len(t, args, 1)
				payload, ok := args[0].(*billingjobs.GenerateInvoicePDFPayload)
				return ok &&
					payload.InvoiceID == invoiceID &&
					payload.BaseURL == "https://billing.example.test" &&
					payload.OrganizationID == orgID &&
					payload.BusinessUnitID == buID &&
					payload.UserID == userID
			}),
		).
		Return(fakeWorkflowRun{id: "wf-invoice-pdf", runID: "run-1"}, nil).
		Once()

	svc := &Service{
		l:               zap.NewNop(),
		workflowStarter: workflowStarter,
	}

	result, err := svc.GeneratePDF(
		t.Context(),
		&servicesports.InvoicePreviewRequest{
			InvoiceID:  invoiceID,
			TenantInfo: tenantInfo,
			BaseURL:    "https://billing.example.test",
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, invoiceID, result.InvoiceID)
	assert.Equal(t, "wf-invoice-pdf", result.WorkflowID)
	assert.Equal(t, "run-1", result.WorkflowRunID)
	assert.Equal(t, "Queued", result.Status)
}

func TestGeneratePDFWorkflowDisabled(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	invoiceID := pulid.MustNew("inv_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	workflowStarter := mocks.NewMockWorkflowStarter(t)
	workflowStarter.EXPECT().Enabled().Return(false).Once()

	svc := &Service{
		l:               zap.NewNop(),
		workflowStarter: workflowStarter,
	}

	result, err := svc.GeneratePDF(
		t.Context(),
		&servicesports.InvoicePreviewRequest{InvoiceID: invoiceID, TenantInfo: tenantInfo},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.Error(t, err)
	require.Nil(t, result)
	assert.ErrorIs(t, err, servicesports.ErrWorkflowStarterDisabled)
}

func TestAutoSendInvoiceAfterPDFGenerationSkipsDisabledAndAlreadySending(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		enabled  bool
		status   invoice.SendStatus
		customer *customer.Customer
	}{
		{
			name:    "disabled billing automation",
			enabled: false,
			status:  invoice.SendStatusNotSent,
		},
		{
			name:    "sending invoice",
			enabled: true,
			status:  invoice.SendStatusSending,
		},
		{
			name:    "sent invoice",
			enabled: true,
			status:  invoice.SendStatusSent,
		},
		{
			name:    "partially sent invoice",
			enabled: true,
			status:  invoice.SendStatusPartiallySent,
		},
		{
			name:    "missing billing profile",
			enabled: false,
			status:  invoice.SendStatusNotSent,
			customer: &customer.Customer{
				ID: pulid.MustNew("cus_"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orgID := pulid.MustNew("org_")
			buID := pulid.MustNew("bu_")
			userID := pulid.MustNew("usr_")
			invoiceID := pulid.MustNew("inv_")
			customerID := pulid.MustNew("cus_")
			tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
			entity := &invoice.Invoice{
				ID:             invoiceID,
				OrganizationID: orgID,
				BusinessUnitID: buID,
				CustomerID:     customerID,
				SendStatus:     tc.status,
			}

			repo := mocks.NewMockInvoiceRepository(t)
			repo.EXPECT().
				GetByID(mock.Anything, repositories.GetInvoiceByIDRequest{
					ID:         invoiceID,
					TenantInfo: tenantInfo,
				}).
				Return(entity, nil).
				Once()

			cus := tc.customer
			if cus == nil {
				cus = &customer.Customer{
					ID: customerID,
					BillingProfile: &customer.CustomerBillingProfile{
						AutoSendInvoiceOnGeneration: tc.enabled,
					},
				}
			} else {
				cus.ID = customerID
			}

			customerRepo := mocks.NewMockCustomerRepository(t)
			customerRepo.EXPECT().
				GetByID(mock.Anything, repositories.GetCustomerByIDRequest{
					ID:         customerID,
					TenantInfo: tenantInfo,
					CustomerFilterOptions: repositories.CustomerFilterOptions{
						IncludeBillingProfile: true,
					},
				}).
				Return(cus, nil).
				Once()

			svc := &Service{
				l:            zap.NewNop(),
				repo:         repo,
				customerRepo: customerRepo,
			}

			result, err := svc.AutoSendInvoiceAfterPDFGeneration(
				t.Context(),
				&servicesports.AutoSendInvoiceAfterPDFGenerationRequest{
					InvoiceID:  invoiceID,
					TenantInfo: tenantInfo,
					BaseURL:    "https://billing.example.test",
				},
				testutil.NewSessionActor(userID, orgID, buID),
			)

			require.NoError(t, err)
			require.Nil(t, result)
		})
	}
}

func TestAutoSendInvoiceAfterPDFGenerationRecordsSendFailure(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	invoiceID := pulid.MustNew("inv_")
	customerID := pulid.MustNew("cus_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
	entity := &invoice.Invoice{
		ID:             invoiceID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CustomerID:     customerID,
		SendStatus:     invoice.SendStatusNotSent,
	}

	repo := mocks.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceByIDRequest{
			ID:         invoiceID,
			TenantInfo: tenantInfo,
		}).
		Return(entity, nil).
		Twice()
	repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(updated *invoice.Invoice) bool {
			return updated.ID == invoiceID &&
				updated.SendStatus == invoice.SendStatusFailed &&
				strings.Contains(updated.LastSendError, "Invoice email delivery is not configured") &&
				updated.SentByID == userID
		})).
		Return(entity, nil).
		Once()

	customerRepo := mocks.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetCustomerByIDRequest{
			ID:         customerID,
			TenantInfo: tenantInfo,
			CustomerFilterOptions: repositories.CustomerFilterOptions{
				IncludeBillingProfile: true,
			},
		}).
		Return(&customer.Customer{
			ID: customerID,
			BillingProfile: &customer.CustomerBillingProfile{
				AutoSendInvoiceOnGeneration: true,
			},
		}, nil).
		Once()

	svc := &Service{
		l:            zap.NewNop(),
		repo:         repo,
		customerRepo: customerRepo,
		auditService: &mocks.NoopAuditService{},
	}

	result, err := svc.AutoSendInvoiceAfterPDFGeneration(
		t.Context(),
		&servicesports.AutoSendInvoiceAfterPDFGenerationRequest{
			InvoiceID:  invoiceID,
			TenantInfo: tenantInfo,
			BaseURL:    "https://billing.example.test",
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "Invoice email delivery is not configured")
}

func testInvoiceForPDF(invoiceID, orgID, buID pulid.ID) *invoice.Invoice {
	return &invoice.Invoice{
		ID:                 invoiceID,
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		BillingQueueItemID: pulid.MustNew("bqi_"),
		ShipmentID:         pulid.MustNew("shp_"),
		CustomerID:         pulid.MustNew("cus_"),
		Number:             "INV-1001",
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusDraft,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		InvoiceDate:        1_700_000_000,
		BillToName:         "Acme Logistics",
		SubtotalAmount:     decimal.NewFromInt(100),
		TotalAmount:        decimal.NewFromInt(100),
	}
}

var pdfPageObjectPattern = regexp.MustCompile(`/Type\s*/Page\b`)

func countPDFPages(content []byte) int {
	return len(pdfPageObjectPattern.FindAll(content, -1))
}

func testInvoicePDFDeliveryProfile() *invoiceDeliveryProfile {
	state := &usstate.UsState{Abbreviation: "IL", CountryName: "USA"}
	pickupDate := int64(1_699_900_000)
	deliveryDate := int64(1_700_100_000)

	return &invoiceDeliveryProfile{
		Organization: &tenant.Organization{
			Name:         "Trenova Logistics",
			AddressLine1: "1 Market Street",
			City:         "Chicago",
			State:        state,
			PostalCode:   "60654",
			DOTNumber:    "1234567",
			ScacCode:     "TRNV",
		},
		BillingControl: &tenant.BillingControl{
			ShowDueDateOnInvoice:    true,
			ShowBalanceDueOnInvoice: true,
			DefaultInvoiceTerms: strings.Join([]string{
				"Carrier agrees that all services are performed subject to the terms and conditions previously executed between Carrier and Customer.",
				"Invoice charges are true and correct and transportation services were performed as described above.",
			}, "\n"),
			DefaultInvoiceFooter: "Thank you for your business. Call (630) 954-0200 or email apinvoices@trenova.com with any questions.",
		},
		Shipment: &shipment.Shipment{
			ProNumber:          "PRO123",
			BOL:                "BOL123",
			ActualShipDate:     &pickupDate,
			ActualDeliveryDate: &deliveryDate,
			Moves: []*shipment.ShipmentMove{
				{
					Stops: []*shipment.Stop{
						{
							Type:     shipment.StopTypePickup,
							Sequence: 1,
							Location: &location.Location{
								Name:         "Origin DC",
								AddressLine1: "100 Pickup Rd",
								City:         "Chicago",
								State:        state,
								PostalCode:   "60654",
							},
						},
						{
							Type:     shipment.StopTypeDelivery,
							Sequence: 2,
							Location: &location.Location{
								Name:         "Destination DC",
								AddressLine1: "200 Delivery Ln",
								City:         "Chicago",
								State:        state,
								PostalCode:   "60654",
							},
						},
					},
				},
			},
			Commodities: []*shipment.ShipmentCommodity{
				{
					Pieces: 20,
					Weight: 40000,
					Commodity: &commodity.Commodity{
						Name:         "Industrial Parts",
						Description:  "SKU 12345",
						FreightClass: commodity.FreightClass125,
					},
				},
			},
		},
	}
}

func TestResolveFromEmailRejectsInvalidCustomerOverride(t *testing.T) {
	t.Parallel()

	profile := &email.Profile{SenderEmail: "billing@example.com"}
	customerProfile := &customer.CustomerEmailProfile{FromEmail: "not-an-email"}

	fromEmail, err := resolveFromEmail(profile, customerProfile)

	require.Error(t, err)
	require.Equal(t, "billing@example.com", fromEmail)
}

func TestAppendShipmentDetailIncludesRouteAndCharges(t *testing.T) {
	t.Parallel()

	amount := decimal.NewFromInt(100)
	entity := &invoice.Invoice{
		ShipmentProNumber: "PRO123",
		ShipmentBOL:       "BOL123",
		CurrencyCode:      "USD",
		ServiceDate:       int64Ptr(1_700_000_000),
		Lines: []*invoice.InoviceLine{
			{
				Description: "Freight",
				Amount:      amount,
			},
		},
	}
	shp := &shipment.Shipment{
		ActualShipDate:     int64Ptr(1_699_900_000),
		ActualDeliveryDate: int64Ptr(1_700_100_000),
		Pieces:             int64Ptr(12),
		Weight:             int64Ptr(24000),
		Moves: []*shipment.ShipmentMove{
			{
				Stops: []*shipment.Stop{
					{
						Type:     shipment.StopTypePickup,
						Sequence: 1,
						Location: &location.Location{
							Name:       "Origin DC",
							City:       "Nashville",
							PostalCode: "37201",
						},
					},
					{
						Type:     shipment.StopTypeDelivery,
						Sequence: 2,
						Location: &location.Location{
							Name:       "Destination DC",
							City:       "Atlanta",
							PostalCode: "30301",
						},
					},
				},
			},
		},
	}

	body := appendShipmentDetail("Please pay.", entity, shp)

	require.Contains(t, body, "Shipment Detail")
	require.Contains(t, body, "Route: Origin DC Nashville 37201 -> Destination DC Atlanta 30301")
	require.Contains(t, body, "Pieces: 12")
	require.Contains(t, body, "- Freight: USD 100.00")
}

func TestShouldAutoPost(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")

	t.Run(
		"organization auto posting requires both org automation and customer opt-in",
		func(t *testing.T) {
			t.Parallel()

			billingRepo := mocks.NewMockBillingControlRepository(t)
			billingRepo.EXPECT().GetByOrgID(t.Context(), orgID).Return(&tenant.BillingControl{
				InvoiceDraftCreationMode: tenant.InvoiceDraftCreationModeAutomaticWhenTransferred,
				InvoicePostingMode:       tenant.InvoicePostingModeAutomaticWhenNoBlockingExceptions,
			}, nil)

			svc := &Service{
				l:           zap.NewNop(),
				billingRepo: billingRepo,
			}

			autoPost := svc.shouldAutoPost(t.Context(), orgID, &customer.Customer{
				BillingProfile: &customer.CustomerBillingProfile{AutoBill: true},
			})

			assert.True(t, autoPost)
		},
	)

	t.Run("customer auto bill cannot loosen organization manual review policy", func(t *testing.T) {
		t.Parallel()

		billingRepo := mocks.NewMockBillingControlRepository(t)
		billingRepo.EXPECT().GetByOrgID(t.Context(), orgID).Return(&tenant.BillingControl{
			InvoicePostingMode: tenant.InvoicePostingModeManualReviewRequired,
		}, nil)

		svc := &Service{
			l:           zap.NewNop(),
			billingRepo: billingRepo,
		}

		autoPost := svc.shouldAutoPost(t.Context(), orgID, &customer.Customer{
			BillingProfile: &customer.CustomerBillingProfile{AutoBill: true},
		})

		assert.False(t, autoPost)
	})

	t.Run("customer fallback still works on tenant not found", func(t *testing.T) {
		t.Parallel()

		billingRepo := mocks.NewMockBillingControlRepository(t)
		billingRepo.EXPECT().GetByOrgID(t.Context(), orgID).
			Return(nil, errortypes.NewNotFoundError("billing control not found"))

		svc := &Service{
			l:           zap.NewNop(),
			billingRepo: billingRepo,
		}

		autoPost := svc.shouldAutoPost(t.Context(), orgID, &customer.Customer{
			BillingProfile: &customer.CustomerBillingProfile{AutoBill: true},
		})

		assert.True(t, autoPost)
	})
}

func TestValidatorValidateCreateRejectsHeaderLineTotalMismatch(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	validator := &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*invoice.Invoice]().Build(),
	}
	entity := &invoice.Invoice{
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		BillingQueueItemID: pulid.MustNew("bqi_"),
		ShipmentID:         pulid.MustNew("shp_"),
		CustomerID:         pulid.MustNew("cus_"),
		Number:             "INV-1001",
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusDraft,
		PaymentTerm:        invoice.PaymentTermNet30,
		CurrencyCode:       "USD",
		InvoiceDate:        1_700_000_000,
		BillToName:         "Acme Logistics",
		SubtotalAmount:     decimal.NewFromInt(100),
		OtherAmount:        decimal.NewFromInt(150),
		TotalAmount:        decimal.NewFromInt(250),
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
		Lines: []*invoice.InoviceLine{
			{
				LineNumber:  1,
				Type:        invoice.InvoiceLineTypeFreight,
				Description: "Freight charge",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(100),
				Amount:      decimal.NewFromInt(100),
			},
			{
				LineNumber:  2,
				Type:        invoice.InvoiceLineTypeAccessorial,
				Description: "Detention",
				Quantity:    decimal.NewFromInt(2),
				UnitPrice:   decimal.RequireFromString("37.50"),
				Amount:      decimal.NewFromInt(75),
			},
		},
	}

	multiErr := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "otherAmount")
	assertErrorField(t, multiErr, "totalAmount")
}

func TestValidatorValidatePost_BlocksLockedPeriodPosting(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	inv := &invoice.Invoice{
		ID:             pulid.MustNew("inv_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ShipmentID:     pulid.MustNew("shp_"),
		BillType:       billingqueue.BillTypeInvoice,
		TotalAmount:    decimal.NewFromInt(100),
	}

	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		AccountingBasis:           tenant.AccountingBasisAccrual,
		RevenueRecognitionPolicy:  tenant.RevenueRecognitionOnInvoicePost,
		LockedPeriodPostingPolicy: tenant.LockedPeriodPostingPolicyBlockSubledgerAllowManualJe,
		ReconciliationMode:        tenant.ReconciliationModeDisabled,
		DefaultRevenueAccountID:   pulid.MustNew("gla_"),
		DefaultARAccountID:        pulid.MustNew("gla_"),
	}, nil)

	fiscalPeriodRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalPeriodRepo.EXPECT().
		GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{
			OrgID: orgID,
			BuID:  buID,
			Date:  1_700_000_000,
		}).
		Return(&fiscalperiod.FiscalPeriod{Status: fiscalperiod.StatusLocked}, nil)

	validator := &Validator{
		l:                zap.NewNop(),
		accountingRepo:   accountingRepo,
		fiscalPeriodRepo: fiscalPeriodRepo,
	}

	multiErr := validator.ValidatePost(t.Context(), inv, pagination.TenantInfo{
		OrgID: orgID,
		BuID:  buID,
	}, 1_700_000_000)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "postedAt")
}

func TestValidatorValidatePostRequiresDefaultLedgerAccounts(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	inv := &invoice.Invoice{
		ID:             pulid.MustNew("inv_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		BillType:       billingqueue.BillTypeInvoice,
	}
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		AccountingBasis:          tenant.AccountingBasisAccrual,
		RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost,
		ReconciliationMode:       tenant.ReconciliationModeDisabled,
	}, nil)
	fiscalPeriodRepo := mocks.NewMockFiscalPeriodRepository(t)
	validator := &Validator{
		l:                zap.NewNop(),
		accountingRepo:   accountingRepo,
		fiscalPeriodRepo: fiscalPeriodRepo,
	}

	multiErr := validator.ValidatePost(t.Context(), inv, pagination.TenantInfo{
		OrgID: orgID,
		BuID:  buID,
	}, 1_700_000_000)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "accountingControl")
}

func TestValidatorValidatePostRequiresFiscalPeriod(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	now := int64(1_700_000_000)
	inv := &invoice.Invoice{
		ID:             pulid.MustNew("inv_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		BillType:       billingqueue.BillTypeInvoice,
	}
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		AccountingBasis:          tenant.AccountingBasisAccrual,
		RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost,
		ReconciliationMode:       tenant.ReconciliationModeDisabled,
		DefaultRevenueAccountID:  pulid.MustNew("gla_"),
		DefaultARAccountID:       pulid.MustNew("gla_"),
	}, nil)
	fiscalPeriodRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalPeriodRepo.EXPECT().
		GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{
			OrgID: orgID,
			BuID:  buID,
			Date:  now,
		}).
		Return(nil, errortypes.NewNotFoundError("fiscal period not found"))
	validator := &Validator{
		l:                zap.NewNop(),
		accountingRepo:   accountingRepo,
		fiscalPeriodRepo: fiscalPeriodRepo,
	}

	multiErr := validator.ValidatePost(t.Context(), inv, pagination.TenantInfo{
		OrgID: orgID,
		BuID:  buID,
	}, now)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "postedAt")
}

func TestInvoicePostingSourceEvent(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		tenant.JournalSourceEventInvoicePosted,
		invoicePostingSourceEvent(billingqueue.BillTypeInvoice),
	)
	assert.Equal(
		t,
		tenant.JournalSourceEventCreditMemoPosted,
		invoicePostingSourceEvent(billingqueue.BillTypeCreditMemo),
	)
	assert.Equal(
		t,
		tenant.JournalSourceEventDebitMemoPosted,
		invoicePostingSourceEvent(billingqueue.BillTypeDebitMemo),
	)
}

func TestInvoicePostingWorkflow(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	now := int64(1_700_000_000)

	entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt := invoicePostingWorkflow(
		&tenant.AccountingControl{
			JournalPostingMode:      tenant.JournalPostingModeAutomatic,
			RequireManualJEApproval: true,
		},
		userID,
		now,
	)
	assert.Equal(t, "Posted", entryStatus)
	assert.Equal(t, "Posted", batchStatus)
	require.NotNil(t, postedAt)
	assert.Equal(t, now, *postedAt)
	assert.Equal(t, userID, postedByID)
	assert.False(t, requiresApproval)
	assert.True(t, isApproved)
	assert.Equal(t, userID, approvedByID)
	require.NotNil(t, approvedAt)

	entryStatus, batchStatus, postedAt, postedByID, requiresApproval, isApproved, approvedByID, approvedAt = invoicePostingWorkflow(
		&tenant.AccountingControl{
			JournalPostingMode:      tenant.JournalPostingModeManual,
			RequireManualJEApproval: true,
		},
		userID,
		now,
	)
	assert.Equal(t, "Pending", entryStatus)
	assert.Equal(t, "Pending", batchStatus)
	assert.Nil(t, postedAt)
	assert.True(t, postedByID.IsNil())
	assert.True(t, requiresApproval)
	assert.False(t, isApproved)
	assert.True(t, approvedByID.IsNil())
	assert.Nil(t, approvedAt)
}

func TestCreateInvoiceJournalPostingBuildsCreditMemoPolarity(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	fyID := pulid.MustNew("fy_")
	periodID := pulid.MustNew("fp_")
	now := int64(1_700_000_000)
	revenueAccountID := pulid.MustNew("gla_")
	arAccountID := pulid.MustNew("gla_")

	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		AccountingBasis:          tenant.AccountingBasisAccrual,
		RevenueRecognitionPolicy: tenant.RevenueRecognitionOnInvoicePost,
		JournalPostingMode:       tenant.JournalPostingModeAutomatic,
		AutoPostSourceEvents: []tenant.JournalSourceEventType{
			tenant.JournalSourceEventCreditMemoPosted,
		},
		DefaultRevenueAccountID: revenueAccountID,
		DefaultARAccountID:      arAccountID,
	}, nil)

	fiscalPeriodRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalPeriodRepo.EXPECT().
		GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: now}).
		Return(&fiscalperiod.FiscalPeriod{
			ID:           periodID,
			FiscalYearID: fyID,
			Status:       fiscalperiod.StatusOpen,
		}, nil)

	journalRepo := mocks.NewMockJournalPostingRepository(t)
	var lastParams *repositories.CreateJournalPostingParams
	journalRepo.EXPECT().
		CreatePosting(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, params repositories.CreateJournalPostingParams) error {
			copyParams := params
			copyParams.Lines = append([]repositories.JournalPostingLine(nil), params.Lines...)
			lastParams = &copyParams
			return nil
		}).
		Once()
	svc := &Service{
		l:                 zap.NewNop(),
		accountingRepo:    accountingRepo,
		journalRepo:       journalRepo,
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		validator:         &Validator{fiscalPeriodRepo: fiscalPeriodRepo},
	}

	err := svc.createInvoiceJournalPosting(t.Context(), &invoice.Invoice{
		ID:               pulid.MustNew("inv_"),
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		CustomerID:       pulid.MustNew("cus_"),
		Number:           "CM-1001",
		BillType:         billingqueue.BillTypeCreditMemo,
		TotalAmountMinor: -13500,
		PostedAt:         &now,
	}, testutil.NewSessionActor(userID, orgID, buID))

	require.NoError(t, err)
	require.NotNil(t, lastParams)
	assert.Equal(t, tenant.JournalSourceEventCreditMemoPosted.String(), lastParams.SourceEventType)
	require.Len(t, lastParams.Lines, 2)
	assert.Equal(t, revenueAccountID, lastParams.Lines[0].GLAccountID)
	assert.Equal(t, int64(13500), lastParams.Lines[0].DebitAmount)
	assert.Equal(t, arAccountID, lastParams.Lines[1].GLAccountID)
	assert.Equal(t, int64(13500), lastParams.Lines[1].CreditAmount)
}

func TestCreateInvoiceJournalPostingSkipsWhenRecognitionPolicyDoesNotAllowInvoicePosting(
	t *testing.T,
) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	now := int64(1_700_000_000)
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		AccountingBasis:          tenant.AccountingBasisCash,
		RevenueRecognitionPolicy: tenant.RevenueRecognitionOnCashReceipt,
		JournalPostingMode:       tenant.JournalPostingModeAutomatic,
		AutoPostSourceEvents: []tenant.JournalSourceEventType{
			tenant.JournalSourceEventCustomerPaymentPosted,
		},
	}, nil)
	journalRepo := mocks.NewMockJournalPostingRepository(t)
	svc := &Service{
		l:                 zap.NewNop(),
		accountingRepo:    accountingRepo,
		journalRepo:       journalRepo,
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		validator:         &Validator{},
	}

	err := svc.createInvoiceJournalPosting(
		t.Context(),
		&invoice.Invoice{
			ID:               pulid.MustNew("inv_"),
			OrganizationID:   orgID,
			BusinessUnitID:   buID,
			BillType:         billingqueue.BillTypeInvoice,
			TotalAmountMinor: 10000,
			PostedAt:         &now,
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.NoError(t, err)
}

func TestCreateInvoiceJournalPostingSkipsWhenAutoPostDisabled(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	now := int64(1_700_000_000)
	accountingRepo := mocks.NewMockAccountingControlRepository(t)
	accountingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(&tenant.AccountingControl{
		JournalPostingMode: tenant.JournalPostingModeAutomatic,
		AutoPostSourceEvents: []tenant.JournalSourceEventType{
			tenant.JournalSourceEventCustomerPaymentPosted,
		},
	}, nil)
	journalRepo := mocks.NewMockJournalPostingRepository(t)
	svc := &Service{
		l:                 zap.NewNop(),
		accountingRepo:    accountingRepo,
		journalRepo:       journalRepo,
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "SEQ-1"},
		validator:         &Validator{},
	}

	err := svc.createInvoiceJournalPosting(
		t.Context(),
		&invoice.Invoice{
			ID:               pulid.MustNew("inv_"),
			OrganizationID:   orgID,
			BusinessUnitID:   buID,
			BillType:         billingqueue.BillTypeInvoice,
			TotalAmountMinor: 10000,
			PostedAt:         &now,
		},
		testutil.NewSessionActor(pulid.MustNew("usr_"), orgID, buID),
	)

	require.NoError(t, err)
}

func TestResolveInvoicePostingPeriodUsesNextOpenPeriod(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	fyID := pulid.MustNew("fy_")
	now := int64(1_700_000_000)
	fiscalRepo := mocks.NewMockFiscalPeriodRepository(t)
	fiscalRepo.EXPECT().
		GetPeriodByDate(mock.Anything, repositories.GetPeriodByDateRequest{OrgID: orgID, BuID: buID, Date: now}).
		Return(&fiscalperiod.FiscalPeriod{FiscalYearID: fyID, PeriodNumber: 1, Status: fiscalperiod.StatusClosed}, nil)
	fiscalRepo.EXPECT().
		ListByFiscalYearID(mock.Anything, repositories.ListByFiscalYearIDRequest{FiscalYearID: fyID, OrgID: orgID, BuID: buID}).
		Return([]*fiscalperiod.FiscalPeriod{{FiscalYearID: fyID, PeriodNumber: 1, Status: fiscalperiod.StatusClosed}, {ID: pulid.MustNew("fp_"), FiscalYearID: fyID, PeriodNumber: 2, Status: fiscalperiod.StatusOpen, StartDate: 1_700_001_000}}, nil)
	svc := &Service{validator: &Validator{fiscalPeriodRepo: fiscalRepo}}

	period, date, err := svc.resolveInvoicePostingPeriod(
		t.Context(),
		&invoice.Invoice{OrganizationID: orgID, BusinessUnitID: buID, PostedAt: &now},
		&tenant.AccountingControl{
			ClosedPeriodPostingPolicy: tenant.ClosedPeriodPostingPolicyPostToNextOpen,
		},
	)

	require.NoError(t, err)
	require.NotNil(t, period)
	assert.Equal(t, int64(1_700_001_000), date)
}

func int64Ptr(v int64) *int64 {
	return &v
}

func testPNGLogo(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, testLogoImage()))
	return buf.Bytes()
}

func testJPEGLogo(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, testLogoImage(), nil))
	return buf.Bytes()
}

func testWebPLogo(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, webp.Encode(&buf, testLogoImage(), nil))
	return buf.Bytes()
}

func testLogoImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 20, G: 80, B: 120, A: 255})
	img.Set(1, 0, color.RGBA{R: 20, G: 80, B: 120, A: 255})
	img.Set(0, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	img.Set(1, 1, color.RGBA{R: 20, G: 80, B: 120, A: 255})
	return img
}

func assertErrorField(t *testing.T, multiErr *errortypes.MultiError, field string) {
	t.Helper()

	require.NotNil(t, multiErr)
	for _, err := range multiErr.Errors {
		if err.Field == field {
			return
		}
	}

	t.Fatalf("expected validation error for field %q, got %#v", field, multiErr.Errors)
}
