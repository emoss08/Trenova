package invoiceservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func amt(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

// parseMoney undoes moneyString, so a test can add up what the document actually
// prints rather than the values it was built from.
func parseAmt(t *testing.T, formatted string) decimal.Decimal {
	t.Helper()
	_, amount, found := strings.Cut(formatted, " ")
	require.True(t, found, "not a money string: %q", formatted)

	return decimal.RequireFromString(amount)
}

func consolidatedInvoice(mutate func(*invoice.Invoice)) *invoice.Invoice {
	shipmentOne := pulid.MustNew("shp_")
	shipmentTwo := pulid.MustNew("shp_")
	periodStart := int64(1_772_000_000)
	periodEnd := int64(1_774_000_000)

	entity := &invoice.Invoice{
		Number:        "INV-9001",
		Scope:         invoice.ScopeConsolidated,
		CurrencyCode:  "USD",
		InvoiceDate:   1_774_000_000,
		ShipmentCount: 2,
		PeriodStart:   &periodStart,
		PeriodEnd:     &periodEnd,
		Detail:        customer.InvoiceDetailDetailed,
		Customer:      &customer.Customer{Name: "Acme Freight", Code: "ACME"},
		SubtotalAmount: amt("400"),
		TotalAmount:    amt("400"),
		Lines: []*invoice.InvoiceLine{
			{
				LineNumber:        1,
				Description:       "Linehaul",
				Amount:            amt("100"),
				ShipmentID:        shipmentOne,
				ShipmentProNumber: "PRO-1",
			},
			{
				LineNumber:        2,
				Description:       "Fuel surcharge",
				Amount:            amt("50"),
				ShipmentID:        shipmentOne,
				ShipmentProNumber: "PRO-1",
			},
			{
				LineNumber:        3,
				Description:       "Linehaul",
				Amount:            amt("200"),
				ShipmentID:        shipmentTwo,
				ShipmentProNumber: "PRO-2",
			},
			{
				LineNumber:  4,
				Description: "Customs brokerage",
				Amount:      amt("50"),
			},
		},
	}
	if mutate != nil {
		mutate(entity)
	}

	return entity
}

func consolidatedProfile(entity *invoice.Invoice) *invoiceDeliveryProfile {
	legs := entity.LegShipmentIDs()
	return &invoiceDeliveryProfile{
		Customer: entity.Customer,
		Shipments: []*repositories.ShipmentSummary{
			{
				ShipmentID:       legs[0],
				ProNumber:        "PRO-1",
				BOL:              "BOL-1",
				PONumber:         "PO-77",
				OriginCity:       "Austin",
				OriginState:      "TX",
				DestinationCity:  "Dallas",
				DestinationState: "TX",
				TotalCharge:      decimal.NewNullDecimal(amt("150")),
			},
			{
				ShipmentID:       legs[1],
				ProNumber:        "PRO-2",
				OriginCity:       "Austin",
				OriginState:      "TX",
				DestinationCity:  "Waco",
				DestinationState: "TX",
				TotalCharge:      decimal.NewNullDecimal(amt("200")),
			},
		},
	}
}

// The live bug this fixes: a grouped invoice has no header shipment, so the
// profile carried none, and every one went out with an empty Shipper, an empty
// Consignee and no freight at all. A manifest replaces them — an invoice that
// bills forty loads has no single shipper, and printing the first one's address
// at the top would state something untrue.
func TestConsolidatedInvoiceListsItsShipmentsInsteadOfFreightDetail(t *testing.T) {
	t.Parallel()

	entity := consolidatedInvoice(nil)
	data := buildInvoicePDFData(entity, consolidatedProfile(entity))

	require.Len(t, data.ShipmentRows, 2)
	assert.Equal(t, "PRO-1", data.ShipmentRows[0].ProNumber)
	assert.Equal(t, "BOL-1", data.ShipmentRows[0].BOL)
	assert.Equal(t, "PO-77", data.ShipmentRows[0].PONumber)
	assert.Equal(t, "Austin, TX", data.ShipmentRows[0].Origin)
	assert.Equal(t, "Dallas, TX", data.ShipmentRows[0].Destination)
	assert.Equal(t, "2", data.ShipmentCount)

	assert.Empty(t, data.Shipper.Name, "no single shipper covers a consolidated invoice")
	assert.Empty(t, data.Consignee.Name)
	assert.Empty(t, data.CommodityRows)
}

func TestConsolidatedInvoiceNamesItsBillingPeriod(t *testing.T) {
	t.Parallel()

	entity := consolidatedInvoice(nil)
	data := buildInvoicePDFData(entity, consolidatedProfile(entity))

	assert.NotEmpty(t, data.Period)
	assert.Contains(t, data.Period, " - ")
}

// A period is only meaningful on an invoice billed on a cycle, and an empty
// "Period:" row on an ordinary invoice is worse than none.
func TestInvoiceWithoutAPeriodSaysNothingAboutOne(t *testing.T) {
	t.Parallel()

	entity := consolidatedInvoice(func(e *invoice.Invoice) {
		e.PeriodStart = nil
		e.PeriodEnd = nil
	})

	assert.Empty(t, buildInvoicePDFData(entity, consolidatedProfile(entity)).Period)
}

// Every ordinary invoice must render exactly as it did: the freight blocks stay,
// the manifest is absent, and nothing repeats the PRO on each charge because the
// header already names it.
func TestSingleShipmentInvoiceKeepsItsFreightDetail(t *testing.T) {
	t.Parallel()

	shp := &shipment.Shipment{
		ID:       pulid.MustNew("shp_"),
		BOL:      "BOL-1",
		Moves:    nil,
		Customer: &customer.Customer{Name: "Acme Freight"},
	}
	entity := &invoice.Invoice{
		Number:        "INV-9002",
		Scope:         invoice.ScopeShipment,
		CurrencyCode:  "USD",
		ShipmentCount: 1,
		ShipmentID:    shp.ID,
		Detail:        customer.InvoiceDetailDetailed,
		Customer:      &customer.Customer{Name: "Acme Freight"},
		Lines: []*invoice.InvoiceLine{
			{
				LineNumber:        1,
				Description:       "Linehaul",
				Amount:            amt("100"),
				ShipmentID:        shp.ID,
				ShipmentProNumber: "PRO-1",
			},
		},
	}

	data := buildInvoicePDFData(entity, &invoiceDeliveryProfile{
		Customer: entity.Customer,
		Shipment: shp,
	})

	assert.Empty(t, data.ShipmentRows)
	assert.Empty(t, data.ShipmentCount)
	require.Len(t, data.ChargeRows, 1)
	assert.Empty(
		t,
		data.ChargeRows[0].ProNumber,
		"the header already names the shipment; repeating it on every row is noise",
	)
}

func TestConsolidatedChargeRowsCarryTheirShipment(t *testing.T) {
	t.Parallel()

	entity := consolidatedInvoice(nil)
	data := buildInvoicePDFData(entity, consolidatedProfile(entity))

	require.Len(t, data.ChargeRows, 4)
	assert.Equal(t, "PRO-1", data.ChargeRows[0].ProNumber)
	assert.Equal(t, "PRO-2", data.ChargeRows[2].ProNumber)
	assert.Empty(
		t,
		data.ChargeRows[3].ProNumber,
		"an order-level charge belongs to no single shipment",
	)
}

// Summary is what a customer asked for when they do not want the accessorial
// breakdown. The per-shipment amounts must still add to the invoice total, or
// the document contradicts its own footer.
func TestSummaryDetailCollapsesToOneRowPerShipment(t *testing.T) {
	t.Parallel()

	entity := consolidatedInvoice(func(e *invoice.Invoice) {
		e.Detail = customer.InvoiceDetailSummary
	})
	data := buildInvoicePDFData(entity, consolidatedProfile(entity))

	require.Len(t, data.ChargeRows, 3, "two shipments plus the order-level charge")
	assert.Equal(t, "PRO-1 (2 charges)", data.ChargeRows[0].Description)
	assert.Equal(t, "PRO-2", data.ChargeRows[1].Description)
	assert.Equal(
		t,
		"Customs brokerage",
		data.ChargeRows[2].Description,
		"an order-level charge keeps its own row rather than folding into a shipment",
	)

	assert.Equal(t, "1", data.ChargeRows[0].Line)
	assert.Equal(t, "3", data.ChargeRows[2].Line)

	total := decimal.Zero
	for _, row := range data.ChargeRows {
		total = total.Add(parseAmt(t, row.Amount))
	}
	assert.True(
		t,
		entity.TotalAmount.Equal(total),
		"summary rows must still add to the invoice total: want %s, got %s",
		entity.TotalAmount,
		total,
	)
}

func TestCityStateReadsAsAPlace(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Austin, TX", cityState("Austin", "TX"))
	assert.Equal(t, "Austin", cityState("Austin", ""))
	assert.Equal(t, "TX", cityState("", "TX"))
	assert.Empty(t, cityState("", ""))
}

// The bug itself lived here. resolveDeliveryProfile read the header ShipmentID,
// which a grouped or consolidated invoice does not have, so it resolved no
// freight at all and every such invoice rendered with empty parties and no
// commodity rows. The lines are the authority on what an invoice covers.
func TestResolveDeliveryProfileFindsFreightWithNoHeaderShipment(t *testing.T) {
	t.Parallel()

	entity := consolidatedInvoice(nil)
	require.True(t, entity.ShipmentID.IsNil(), "a consolidated invoice has no header shipment")

	legs := entity.LegShipmentIDs()
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		ListSummariesByIDs(mock.Anything, mock.MatchedBy(
			func(req *repositories.ListShipmentSummariesRequest) bool {
				return len(req.ShipmentIDs) == len(legs)
			},
		)).
		Return([]*repositories.ShipmentSummary{
			{ShipmentID: legs[0], ProNumber: "PRO-1"},
			{ShipmentID: legs[1], ProNumber: "PRO-2"},
		}, nil)

	profile, err := resolveDeliveryProfileWith(
		t.Context(),
		deliveryProfileRepos{shipmentRepo: shipmentRepo},
		resolveDeliveryProfileParams{Entity: entity, IncludeShipmentDetails: true},
	)

	require.NoError(t, err)
	assert.Len(t, profile.Shipments, 2)
	assert.Nil(t, profile.Shipment, "no single shipment covers a consolidated invoice")
}

// A single-shipment invoice still loads the whole shipment, because the email
// context and the freight detail both need its stops and commodities.
func TestResolveDeliveryProfileLoadsTheWholeShipmentWhenThereIsOne(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	entity := &invoice.Invoice{
		ShipmentID: shipmentID,
		Lines: []*invoice.InvoiceLine{
			{LineNumber: 1, ShipmentID: shipmentID, Amount: amt("100")},
		},
	}

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: shipmentID, BOL: "BOL-1"}, nil)

	profile, err := resolveDeliveryProfileWith(
		t.Context(),
		deliveryProfileRepos{shipmentRepo: shipmentRepo},
		resolveDeliveryProfileParams{Entity: entity, IncludeShipmentDetails: true},
	)

	require.NoError(t, err)
	require.NotNil(t, profile.Shipment)
	assert.Equal(t, "BOL-1", profile.Shipment.BOL)
	assert.Empty(t, profile.Shipments)
}
