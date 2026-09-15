package invoiceservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconciliationExpectedTotalUsesThePayersShareOnASplitInvoice(t *testing.T) {
	t.Parallel()

	leg := builderLeg("PRO-1", 1000)
	amd := pulid.MustNew("cus_")
	freightSplit(leg, leg.CustomerID, amd, 60)

	entity := &invoice.Invoice{
		Scope:       invoice.ScopeShipment,
		BillType:    billingqueue.BillTypeInvoice,
		CustomerID:  amd,
		IsSplitBill: true,
		ShipmentID:  leg.ID,
	}

	expected := reconciliationExpectedTotal(entity, []*shipment.Shipment{leg})
	assert.True(t, expected.Equal(decimal.NewFromInt(400)), "AMD owes 40% of the leg, not all of it")

	entity.BillType = billingqueue.BillTypeCreditMemo
	assert.True(t, reconciliationExpectedTotal(entity, []*shipment.Shipment{leg}).Equal(decimal.NewFromInt(-400)))
}

func TestReconciliationExpectedTotalUsesTheLegTotalOnAnOrdinaryInvoice(t *testing.T) {
	t.Parallel()

	leg := builderLeg("PRO-1", 1000)
	amd := pulid.MustNew("cus_")
	freightSplit(leg, leg.CustomerID, amd, 60)

	entity := &invoice.Invoice{
		Scope:      invoice.ScopeShipment,
		BillType:   billingqueue.BillTypeInvoice,
		CustomerID: leg.CustomerID,
		ShipmentID: leg.ID,
	}

	// Not flagged as split, so the whole leg is expected however its rows are allocated.
	assert.True(t, reconciliationExpectedTotal(entity, []*shipment.Shipment{leg}).Equal(decimal.NewFromInt(1000)))
}

func TestReconciliationLegTotalFallsBackWhenThePayerHasNoShare(t *testing.T) {
	t.Parallel()

	leg := builderLeg("PRO-1", 1000)
	entity := &invoice.Invoice{CustomerID: pulid.MustNew("cus_"), IsSplitBill: true}

	assert.True(t, reconciliationLegTotal(entity, leg).IsZero())
	assert.True(t, reconciliationLegTotal(entity, nil).IsZero())
}

func TestReconciliationExpectedTotalAddsUnattributedLinesOnWiderScopes(t *testing.T) {
	t.Parallel()

	leg := builderLeg("PRO-1", 1000)
	amd := pulid.MustNew("cus_")
	freightSplit(leg, leg.CustomerID, amd, 60)

	entity := &invoice.Invoice{
		Scope:       invoice.ScopeOrder,
		BillType:    billingqueue.BillTypeInvoice,
		CustomerID:  amd,
		IsSplitBill: true,
		Lines: []*invoice.InvoiceLine{
			{ShipmentID: leg.ID, Amount: decimal.NewFromInt(400)},
			{Amount: decimal.NewFromInt(25)},
		},
	}
	require.True(t, reconciliationExpectedTotal(entity, []*shipment.Shipment{leg}).Equal(decimal.NewFromInt(425)))
}
