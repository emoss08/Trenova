package invoiceservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func builderLeg(pro string, freight int64) *shipment.Shipment {
	serviceDate := int64(1_700_000_000)
	return &shipment.Shipment{
		ID:                  pulid.MustNew("shp_"),
		CustomerID:          pulid.MustNew("cus_"),
		ProNumber:           pro,
		BOL:                 "BOL-" + pro,
		ActualDeliveryDate:  &serviceDate,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(freight)),
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(freight)),
	}
}

func builderParams(scope invoice.Scope, legs ...*shipment.Shipment) *buildInvoiceParams {
	return &buildInvoiceParams{
		Anchor: &billingqueue.BillingQueueItem{
			ID:             pulid.MustNew("bqi_"),
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			Number:         "INV-1",
			BillType:       billingqueue.BillTypeInvoice,
		},
		Scope:    scope,
		Customer: &customer.Customer{ID: pulid.MustNew("cus_"), Name: "Acme Foods"},
		Control:  &tenant.BillingControl{DefaultPaymentTerm: tenant.PaymentTermNet30},
		Legs:     legs,
	}
}

func TestBuildInvoiceEntityShipmentScope(t *testing.T) {
	t.Parallel()

	leg := builderLeg("PRO-1", 100)
	entity := (&Service{l: zap.NewNop()}).buildInvoiceEntity(
		builderParams(invoice.ScopeShipment, leg),
	)

	require.NotNil(t, entity)
	assert.Equal(t, invoice.ScopeShipment, entity.Scope)
	assert.Equal(t, leg.ID, entity.ShipmentID)
	assert.Equal(t, "PRO-1", entity.ShipmentProNumber)
	assert.Equal(t, 1, entity.ShipmentCount)
	assert.Nil(t, entity.PeriodStart)
}

func TestBuildInvoiceEntityOrderScope(t *testing.T) {
	t.Parallel()

	legs := []*shipment.Shipment{builderLeg("PRO-1", 100), builderLeg("PRO-2", 250)}
	params := builderParams(invoice.ScopeOrder, legs...)
	params.Order = &order.Order{ID: pulid.MustNew("ord_"), OrderNumber: "ORD-9"}

	entity := (&Service{l: zap.NewNop()}).buildInvoiceEntity(params)

	require.NotNil(t, entity)
	assert.Equal(t, invoice.ScopeOrder, entity.Scope)
	assert.Equal(t, params.Order.ID, entity.OrderID)
	assert.Equal(t, "ORD-9", entity.OrderNumber)
	// An order invoice deliberately carries no header shipment — the lines hold
	// the attribution.
	assert.True(t, entity.ShipmentID.IsNil())
	assert.Empty(t, entity.ShipmentProNumber)
	assert.Equal(t, 2, entity.ShipmentCount)
}

func TestBuildInvoiceEntityConsolidatedScope(t *testing.T) {
	t.Parallel()

	start := int64(1_700_000_000)
	end := start + 30*86_400
	runID := pulid.MustNew("invrun_")

	legs := []*shipment.Shipment{
		builderLeg("PRO-1", 100),
		builderLeg("PRO-2", 250),
		builderLeg("PRO-3", 400),
	}
	params := builderParams(invoice.ScopeConsolidated, legs...)
	params.RunID = runID
	params.PeriodStart = &start
	params.PeriodEnd = &end

	entity := (&Service{l: zap.NewNop()}).buildInvoiceEntity(params)

	require.NotNil(t, entity)
	assert.Equal(t, invoice.ScopeConsolidated, entity.Scope)
	// Neither a shipment nor an order: the period and the lines are its identity.
	assert.True(t, entity.ShipmentID.IsNil())
	assert.True(t, entity.OrderID.IsNil())
	assert.Equal(t, runID, entity.InvoiceRunID)
	require.NotNil(t, entity.PeriodStart)
	assert.Equal(t, start, *entity.PeriodStart)
	assert.Equal(t, 3, entity.ShipmentCount)
	assert.Len(t, entity.LegShipmentIDs(), 3)
}

func TestBuildInvoiceEntityNumbersLinesContinuouslyAcrossLegs(t *testing.T) {
	t.Parallel()

	legs := []*shipment.Shipment{
		builderLeg("PRO-1", 100),
		builderLeg("PRO-2", 250),
		builderLeg("PRO-3", 400),
	}
	entity := (&Service{l: zap.NewNop()}).buildInvoiceEntity(
		builderParams(invoice.ScopeConsolidated, legs...),
	)

	require.NotNil(t, entity)
	require.Len(t, entity.Lines, 3)
	for i, line := range entity.Lines {
		assert.Equal(t, i+1, line.LineNumber)
		assert.False(t, line.ShipmentID.IsNil(), "every leg line keeps its attribution")
	}
}

func TestBuildInvoiceEntityOrderChargesCarryNoAttribution(t *testing.T) {
	t.Parallel()

	params := builderParams(invoice.ScopeOrder, builderLeg("PRO-1", 100))
	params.Order = &order.Order{ID: pulid.MustNew("ord_"), OrderNumber: "ORD-9"}
	params.OrderCharges = []*order.OrderCharge{
		{Description: "Customs brokerage", Amount: decimal.NewFromInt(75)},
	}

	entity := (&Service{l: zap.NewNop()}).buildInvoiceEntity(params)

	require.NotNil(t, entity)
	require.Len(t, entity.Lines, 2)
	// The order charge is what lands in the trailing section when rendered, so it
	// must stay unattributed.
	assert.True(t, entity.Lines[1].ShipmentID.IsNil())
	assert.Equal(t, 2, entity.Lines[1].LineNumber)
}

func TestBuildInvoiceEntityNoLegs(t *testing.T) {
	t.Parallel()

	assert.Nil(
		t,
		(&Service{l: zap.NewNop()}).buildInvoiceEntity(builderParams(invoice.ScopeConsolidated)),
	)
}

func TestLegServiceWindow(t *testing.T) {
	t.Parallel()

	early := int64(1_700_000_000)
	late := early + 10*86_400

	legs := []*shipment.Shipment{
		{ActualDeliveryDate: &late},
		{ActualDeliveryDate: &early},
	}
	start, end := legServiceWindow(legs)
	assert.Equal(t, early, start)
	assert.Equal(t, late, end)

	// A selection with no service dates still has to state a period, so it falls
	// back to something ordered rather than leaving the bounds equal.
	undated := []*shipment.Shipment{{CreatedAt: early}}
	start, end = legServiceWindow(undated)
	assert.Equal(t, early, start)
	assert.Greater(t, end, start)
}
