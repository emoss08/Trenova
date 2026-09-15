package invoicelines_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/invoicelines"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// splitEverything divides the freight and every accessorial between payer and
// a second party at percent / (100 - percent).
func splitEverything(shp *shipment.Shipment, payer pulid.ID, percent string) {
	other := pulid.MustNew("cus_")
	remainder := decimal.NewFromInt(100).Sub(dec(percent))

	row := func(kind shipment.ChargeAllocationKind, chargeID *pulid.ID, who pulid.ID, pct decimal.Decimal, seq int16) *shipment.ChargeAllocation {
		return &shipment.ChargeAllocation{
			ID:                 pulid.MustNew("chal_"),
			ChargeKind:         kind,
			AdditionalChargeID: chargeID,
			BillToCustomerID:   who,
			Method:             shipment.ChargeAllocationMethodPercent,
			Percent:            decimal.NewNullDecimal(pct),
			Sequence:           seq,
		}
	}

	shp.ChargeAllocations = append(shp.ChargeAllocations,
		row(shipment.ChargeAllocationKindFreight, nil, payer, dec(percent), 0),
		row(shipment.ChargeAllocationKindFreight, nil, other, remainder, 1),
	)
	for _, charge := range shp.AdditionalCharges {
		id := charge.ID
		shp.ChargeAllocations = append(shp.ChargeAllocations,
			row(shipment.ChargeAllocationKindAccessorial, &id, payer, dec(percent), 0),
			row(shipment.ChargeAllocationKindAccessorial, &id, other, remainder, 1),
		)
	}
}

func TestForShipmentSharePartialFreightCarriesItsShare(t *testing.T) {
	t.Parallel()

	fuel := &accessorialcharge.AccessorialCharge{
		ID:          pulid.MustNew("acc_"),
		Code:        "FSC",
		Description: "Fuel Surcharge",
		Method:      accessorialcharge.MethodPercentage,
	}
	shp := testShipment(&shipment.AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		AccessorialChargeID: fuel.ID,
		AccessorialCharge:   fuel,
		Method:              accessorialcharge.MethodPercentage,
		Amount:              dec("10"),
		Unit:                1,
	})
	shp.CustomerID = pulid.MustNew("cus_")
	splitEverything(shp, shp.CustomerID, "60")

	resolution, err := shipment.ResolveShares(shp, shp.ChargeAllocations)
	require.NoError(t, err)
	share := resolution.ShareFor(shp.CustomerID)
	require.NotNil(t, share)

	lines := invoicelines.ForShipmentShare(billingqueue.BillTypeInvoice, shp, share, 1)
	require.Len(t, lines, 2)

	freight := lines[0]
	assert.Equal(t, "Freight charge (60% share)", freight.Description)
	assert.True(t, freight.Amount.Equal(dec("1470")), "60% of 2450")
	assert.True(t, freight.UnitPrice.Equal(dec("1470")))
	require.True(t, freight.AllocationPercent.Valid)
	assert.True(t, freight.AllocationPercent.Decimal.Equal(dec("60")))
	assert.True(t, freight.ChargeAllocationID.IsNotNil())
	assert.True(t, freight.Rate.Decimal.Equal(dec("3.5")), "the rate is the shipment's, not the share's")
	assert.True(t, freight.IsPartialShare())

	surcharge := lines[1]
	assert.Equal(t, "Fuel Surcharge (60% share)", surcharge.Description)
	assert.True(t, surcharge.Amount.Equal(dec("147")), "60% of the 245 surcharge on the full freight")
	require.True(t, surcharge.RateBasisAmount.Valid)
	assert.True(t, surcharge.RateBasisAmount.Decimal.Equal(dec("2450")), "the basis stays the whole freight")
	assert.True(t, surcharge.AllocationPercent.Valid)
	assert.True(t, surcharge.ChargeAllocationID.IsNotNil())
}

func TestForShipmentShareFullShareMatchesForShipment(t *testing.T) {
	t.Parallel()

	det := detention()
	shp := testShipment(&shipment.AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		AccessorialChargeID: det.ID,
		AccessorialCharge:   det,
		Method:              accessorialcharge.MethodPerUnit,
		Amount:              dec("75"),
		Unit:                2,
	})
	shp.CustomerID = pulid.MustNew("cus_")

	resolution, err := shipment.ResolveShares(shp, nil)
	require.NoError(t, err)

	viaShare := invoicelines.ForShipmentShare(billingqueue.BillTypeInvoice, shp, resolution.Shares[0], 1)
	direct := invoicelines.ForShipment(billingqueue.BillTypeInvoice, shp, 1)
	require.Len(t, viaShare, len(direct))
	for i := range direct {
		assert.Equal(t, direct[i].Description, viaShare[i].Description)
		assert.True(t, direct[i].Amount.Equal(viaShare[i].Amount))
		assert.Equal(t, direct[i].LineNumber, viaShare[i].LineNumber)
		assert.False(t, viaShare[i].AllocationPercent.Valid, "a whole charge carries no share")
		assert.True(t, viaShare[i].ChargeAllocationID.IsNil())
		assert.False(t, viaShare[i].IsPartialShare())
	}
	assert.Equal(t, invoicelines.FreightDescription, viaShare[0].Description)
	assert.Equal(t, "Detention", viaShare[1].Description)
}

func TestForShipmentShareNegatesCreditMemoPartialShares(t *testing.T) {
	t.Parallel()

	shp := testShipment()
	shp.CustomerID = pulid.MustNew("cus_")
	splitEverything(shp, shp.CustomerID, "25")

	resolution, err := shipment.ResolveShares(shp, shp.ChargeAllocations)
	require.NoError(t, err)

	lines := invoicelines.ForShipmentShare(
		billingqueue.BillTypeCreditMemo,
		shp,
		resolution.ShareFor(shp.CustomerID),
		1,
	)
	require.Len(t, lines, 1)
	assert.True(t, lines[0].Amount.Equal(dec("-612.5")))
	assert.True(t, lines[0].UnitPrice.Equal(dec("-612.5")))
	assert.True(t, lines[0].AllocationPercent.Decimal.Equal(dec("25")))
	assert.True(t, lines[0].IsPartialShare())
}

func TestForShipmentShareWithNilShipmentOrEmptyShare(t *testing.T) {
	t.Parallel()

	assert.Nil(t, invoicelines.ForShipmentShare(billingqueue.BillTypeInvoice, nil, nil, 1))

	shp := testShipment()
	lines := invoicelines.ForShipmentShare(
		billingqueue.BillTypeInvoice,
		shp,
		&shipment.PayerShare{PayerID: pulid.MustNew("cus_")},
		1,
	)
	assert.Empty(t, lines, "a payer with no charges gets no lines")
}

func TestForOrderChargeShare(t *testing.T) {
	t.Parallel()

	chargeID := pulid.MustNew("ordchg_")
	allocationID := pulid.MustNew("chal_")

	whole := invoicelines.ForOrderChargeShare(billingqueue.BillTypeInvoice, shipment.AllocatedCharge{
		Kind:          shipment.ChargeAllocationKindOrderCharge,
		OrderChargeID: chargeID,
		Description:   "Customs brokerage",
		ChargeTotal:   dec("250"),
		Amount:        dec("250"),
	}, 7)
	assert.Equal(t, 7, whole.LineNumber)
	assert.Equal(t, invoice.InvoiceLineTypeAccessorial, whole.Type)
	assert.Equal(t, "Customs brokerage", whole.Description)
	assert.True(t, whole.Amount.Equal(dec("250")))
	assert.True(t, whole.UnitPrice.Equal(dec("250")))
	assert.True(t, whole.Quantity.Equal(dec("1")))
	assert.True(t, whole.ShipmentID.IsNil(), "order charges carry no leg attribution")
	assert.False(t, whole.AllocationPercent.Valid)

	partial := invoicelines.ForOrderChargeShare(billingqueue.BillTypeCreditMemo, shipment.AllocatedCharge{
		Kind:          shipment.ChargeAllocationKindOrderCharge,
		OrderChargeID: chargeID,
		AllocationID:  allocationID,
		Description:   "Customs brokerage",
		ChargeTotal:   dec("250"),
		Amount:        dec("100"),
		Percent:       decimal.NewNullDecimal(dec("40")),
		Partial:       true,
	}, 8)
	assert.Equal(t, "Customs brokerage (40% share)", partial.Description)
	assert.True(t, partial.Amount.Equal(dec("-100")))
	assert.Equal(t, allocationID, partial.ChargeAllocationID)
	assert.True(t, partial.AllocationPercent.Decimal.Equal(dec("40")))
}

func TestShareDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		charge shipment.AllocatedCharge
		want   string
	}{
		{
			name:   "whole charge keeps the plain description",
			charge: shipment.AllocatedCharge{Percent: decimal.NewNullDecimal(dec("100"))},
			want:   "Freight charge",
		},
		{
			name:   "trailing zeros are trimmed",
			charge: shipment.AllocatedCharge{Partial: true, Percent: decimal.NewNullDecimal(dec("33.50"))},
			want:   "Freight charge (33.5% share)",
		},
		{
			name:   "whole percents read without decimals",
			charge: shipment.AllocatedCharge{Partial: true, Percent: decimal.NewNullDecimal(dec("60.00"))},
			want:   "Freight charge (60% share)",
		},
		{
			name:   "amount shares without a percent say partial",
			charge: shipment.AllocatedCharge{Partial: true},
			want:   "Freight charge (partial share)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, invoicelines.ShareDescription("Freight charge", tc.charge))
		})
	}
}
