package invoiceservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func freightSplit(leg *shipment.Shipment, payer, other pulid.ID, payerPercent int64) {
	leg.ChargeAllocations = []*shipment.ChargeAllocation{
		{
			ID:               pulid.MustNew("chal_"),
			ChargeKind:       shipment.ChargeAllocationKindFreight,
			BillToCustomerID: payer,
			Method:           shipment.ChargeAllocationMethodPercent,
			Percent:          decimal.NewNullDecimal(decimal.NewFromInt(payerPercent)),
		},
		{
			ID:               pulid.MustNew("chal_"),
			ChargeKind:       shipment.ChargeAllocationKindFreight,
			BillToCustomerID: other,
			Method:           shipment.ChargeAllocationMethodPercent,
			Percent:          decimal.NewNullDecimal(decimal.NewFromInt(100 - payerPercent)),
			Sequence:         1,
		},
	}
}

func TestBuildInvoiceEntityStampsTheShipperAndSplitOnAShipmentInvoice(t *testing.T) {
	t.Parallel()

	leg := builderLeg("PRO-1", 1000)
	amd := pulid.MustNew("cus_")
	freightSplit(leg, leg.CustomerID, amd, 60)

	resolution, err := shipment.ResolveShares(leg, leg.ChargeAllocations)
	require.NoError(t, err)

	params := builderParams(invoice.ScopeShipment, leg)
	params.Customer = &customer.Customer{ID: amd, Name: "AMD"}
	params.Shipper = &customer.Customer{ID: leg.CustomerID, Name: "Intel"}
	params.Shares = map[pulid.ID]*shipment.PayerShare{leg.ID: resolution.ShareFor(amd)}
	params.IsSplitBill = resolution.IsSplit

	entity := (&Service{l: zap.NewNop()}).buildInvoiceEntity(params)

	require.NotNil(t, entity)
	assert.Equal(t, amd, entity.CustomerID, "the payer is the invoice's customer")
	assert.Equal(t, leg.CustomerID, entity.ShipperCustomerID)
	assert.True(t, entity.IsSplitBill)
	assert.Equal(t, "AMD", entity.BillToName)
	require.Len(t, entity.Lines, 1)
	assert.Equal(t, "Freight charge (40% share)", entity.Lines[0].Description)
	assert.True(t, entity.TotalAmount.Equal(decimal.NewFromInt(400)), "40% of 1000")
	assert.True(t, entity.SubtotalAmount.Equal(decimal.NewFromInt(400)))
	assert.Equal(t, int64(40_000), entity.TotalAmountMinor)
}

func TestBuildInvoiceEntityLeavesTheShipperOffAConsolidatedInvoice(t *testing.T) {
	t.Parallel()

	legA := builderLeg("PRO-1", 100)
	legB := builderLeg("PRO-2", 300)
	payer := pulid.MustNew("cus_")
	freightSplit(legA, legA.CustomerID, payer, 50)
	freightSplit(legB, legB.CustomerID, payer, 50)

	shares := make(map[pulid.ID]*shipment.PayerShare, 2)
	for _, leg := range []*shipment.Shipment{legA, legB} {
		resolution, err := shipment.ResolveShares(leg, leg.ChargeAllocations)
		require.NoError(t, err)
		shares[leg.ID] = resolution.ShareFor(payer)
	}

	start := int64(1_700_000_000)
	end := start + 30*86_400
	params := builderParams(invoice.ScopeConsolidated, legA, legB)
	params.Customer = &customer.Customer{ID: payer, Name: "AMD"}
	params.Shipper = &customer.Customer{ID: legA.CustomerID, Name: "Intel"}
	params.Shares = shares
	params.IsSplitBill = true
	params.PeriodStart = &start
	params.PeriodEnd = &end

	entity := (&Service{l: zap.NewNop()}).buildInvoiceEntity(params)

	require.NotNil(t, entity)
	assert.True(t, entity.ShipperCustomerID.IsNil(), "a statement spans shippers; the lines carry it")
	assert.True(t, entity.IsSplitBill)
	assert.True(t, entity.TotalAmount.Equal(decimal.NewFromInt(200)), "half of 100 plus half of 300")
	require.Len(t, entity.Lines, 2)
	for _, line := range entity.Lines {
		assert.True(t, line.IsPartialShare())
	}
}

func TestBuildInvoiceEntityIgnoresAShipperWhoIsThePayer(t *testing.T) {
	t.Parallel()

	leg := builderLeg("PRO-1", 100)
	params := builderParams(invoice.ScopeShipment, leg)
	params.Shipper = params.Customer

	entity := (&Service{l: zap.NewNop()}).buildInvoiceEntity(params)

	require.NotNil(t, entity)
	assert.True(t, entity.ShipperCustomerID.IsNil())
	assert.False(t, entity.IsSplitBill)
}

func TestBuildInvoiceEntityBillsTheOrderChargeShare(t *testing.T) {
	t.Parallel()

	leg := builderLeg("PRO-1", 100)
	params := builderParams(invoice.ScopeOrder, leg)
	params.Order = &order.Order{ID: pulid.MustNew("ord_"), OrderNumber: "ORD-9"}
	params.OrderChargeShare = &shipment.PayerShare{
		PayerID: params.Customer.ID,
		Charges: []shipment.AllocatedCharge{
			{
				Kind:          shipment.ChargeAllocationKindOrderCharge,
				OrderChargeID: pulid.MustNew("ordchg_"),
				AllocationID:  pulid.MustNew("chal_"),
				Description:   "Customs brokerage",
				ChargeTotal:   decimal.NewFromInt(200),
				Amount:        decimal.NewFromInt(50),
				Percent:       decimal.NewNullDecimal(decimal.NewFromInt(25)),
				Partial:       true,
			},
		},
	}
	// Legacy order charges are ignored once a share is supplied.
	params.OrderCharges = []*order.OrderCharge{{Description: "Ignored", Amount: decimal.NewFromInt(999)}}

	entity := (&Service{l: zap.NewNop()}).buildInvoiceEntity(params)

	require.NotNil(t, entity)
	require.Len(t, entity.Lines, 2)
	assert.Equal(t, "Customs brokerage (25% share)", entity.Lines[1].Description)
	assert.True(t, entity.Lines[1].ShipmentID.IsNil())
	assert.True(t, entity.Lines[1].Amount.Equal(decimal.NewFromInt(50)))
	assert.True(t, entity.OtherAmount.Equal(decimal.NewFromInt(50)))
	assert.True(t, entity.TotalAmount.Equal(decimal.NewFromInt(150)))
}

func TestOrderChargeShareFor(t *testing.T) {
	t.Parallel()

	defaultPayer := pulid.MustNew("cus_")
	other := pulid.MustNew("cus_")
	fuelID := pulid.MustNew("ordchg_")
	charges := []*order.OrderCharge{
		{ID: pulid.MustNew("ordchg_"), Description: "Customs brokerage", Amount: decimal.NewFromInt(250)},
		{
			ID:          fuelID,
			Description: "Order fuel",
			Amount:      decimal.NewFromInt(80),
			Allocations: []*shipment.ChargeAllocation{{
				ID:               pulid.MustNew("chal_"),
				ChargeKind:       shipment.ChargeAllocationKindOrderCharge,
				OrderChargeID:    &fuelID,
				BillToCustomerID: other,
				Method:           shipment.ChargeAllocationMethodPercent,
				Percent:          decimal.NewNullDecimal(decimal.NewFromInt(100)),
			}},
		},
	}

	t.Run("nothing to bill yields no share", func(t *testing.T) {
		t.Parallel()

		share, split, err := orderChargeShareFor(nil, defaultPayer, defaultPayer)
		require.NoError(t, err)
		assert.Nil(t, share)
		assert.False(t, split)

		share, split, err = orderChargeShareFor(charges, defaultPayer, pulid.MustNew("cus_"))
		require.NoError(t, err)
		assert.Nil(t, share, "a payer with no share of any charge gets none")
		assert.True(t, split)
	})

	t.Run("the default payer carries the unallocated charge only", func(t *testing.T) {
		t.Parallel()

		share, split, err := orderChargeShareFor(charges, defaultPayer, defaultPayer)
		require.NoError(t, err)
		require.NotNil(t, share)
		assert.True(t, split)
		require.Len(t, share.Charges, 1)
		assert.Equal(t, "Customs brokerage", share.Charges[0].Description)
		assert.True(t, share.Charges[0].AllocationID.IsNil())
	})

	t.Run("the other payer carries their allocated charge", func(t *testing.T) {
		t.Parallel()

		share, split, err := orderChargeShareFor(charges, defaultPayer, other)
		require.NoError(t, err)
		require.NotNil(t, share)
		assert.True(t, split)
		require.Len(t, share.Charges, 1)
		assert.Equal(t, fuelID, share.Charges[0].OrderChargeID)
		assert.True(t, share.Charges[0].Amount.Equal(decimal.NewFromInt(80)))
	})

	t.Run("a bad allocation surfaces as a validation error", func(t *testing.T) {
		t.Parallel()

		bad := &order.OrderCharge{
			ID:          fuelID,
			Description: "Order fuel",
			Amount:      decimal.NewFromInt(80),
			Allocations: []*shipment.ChargeAllocation{{
				ID:               pulid.MustNew("chal_"),
				ChargeKind:       shipment.ChargeAllocationKindOrderCharge,
				OrderChargeID:    &fuelID,
				BillToCustomerID: other,
				Method:           shipment.ChargeAllocationMethodAmount,
				Amount:           decimal.NewNullDecimal(decimal.NewFromInt(79)),
			}},
		}
		_, _, err := orderChargeShareFor([]*order.OrderCharge{bad}, defaultPayer, other)
		require.Error(t, err)
		var typed *errortypes.Error
		require.True(t, errors.As(err, &typed))
		assert.Equal(t, "charges[0].allocations", typed.Field)
	})
}

func TestBucketLegsByPayer(t *testing.T) {
	t.Parallel()

	ord := &order.Order{ID: pulid.MustNew("ord_"), CustomerID: pulid.MustNew("cus_")}
	payerZ := pulid.ID("cus_ZZZ")
	payerA := pulid.ID("cus_AAA")

	legOne := builderLeg("PRO-1", 100)
	legOne.CustomerID = ord.CustomerID
	freightSplit(legOne, ord.CustomerID, payerZ, 50)

	legTwo := builderLeg("PRO-2", 300)
	legTwo.CustomerID = ord.CustomerID
	legTwo.BillToCustomerID = &payerA

	buckets, err := bucketLegsByPayer(ord, []*shipment.Shipment{legOne, legTwo})
	require.NoError(t, err)

	require.Len(t, buckets, 3)
	assert.Equal(t, ord.CustomerID, buckets[0].PayerID, "the order customer leads")
	assert.Equal(t, payerA, buckets[1].PayerID)
	assert.Equal(t, payerZ, buckets[2].PayerID)

	require.Len(t, buckets[0].Legs, 1)
	assert.Equal(t, legOne.ID, buckets[0].Legs[0].ID)
	assert.True(t, buckets[0].IsSplit)
	assert.True(t, buckets[0].Shares[legOne.ID].TotalAmount.Equal(decimal.NewFromInt(50)))

	require.Len(t, buckets[1].Legs, 1)
	assert.Equal(t, legTwo.ID, buckets[1].Legs[0].ID)
	assert.False(t, buckets[1].IsSplit, "a leg billed whole to its bill-to is not split")
	assert.True(t, buckets[1].Shares[legTwo.ID].TotalAmount.Equal(decimal.NewFromInt(300)))

	require.Len(t, buckets[2].Legs, 1)
	assert.True(t, buckets[2].Shares[legOne.ID].TotalAmount.Equal(decimal.NewFromInt(50)))
}

func TestBucketLegsByPayerWithoutTheOrderCustomerLeadsWithTheFirstPayer(t *testing.T) {
	t.Parallel()

	ord := &order.Order{ID: pulid.MustNew("ord_"), CustomerID: pulid.MustNew("cus_")}
	billTo := pulid.ID("cus_BBB")
	leg := builderLeg("PRO-1", 100)
	leg.CustomerID = ord.CustomerID
	leg.BillToCustomerID = &billTo

	buckets, err := bucketLegsByPayer(ord, []*shipment.Shipment{leg})
	require.NoError(t, err)

	require.Len(t, buckets, 1)
	assert.Equal(t, billTo, buckets[0].PayerID)
}

func TestPayerSharesForLegs(t *testing.T) {
	t.Parallel()

	legA := builderLeg("PRO-1", 100)
	legB := builderLeg("PRO-2", 300)
	payer := pulid.MustNew("cus_")
	freightSplit(legA, legA.CustomerID, payer, 70)
	legB.BillToCustomerID = &payer

	shares, isSplit, err := payerSharesForLegs([]*shipment.Shipment{legA, nil, legB}, payer)
	require.NoError(t, err)
	assert.True(t, isSplit)
	require.Len(t, shares, 2)
	assert.True(t, shares[legA.ID].TotalAmount.Equal(decimal.NewFromInt(30)))
	assert.True(t, shares[legB.ID].TotalAmount.Equal(decimal.NewFromInt(300)))

	_, _, err = payerSharesForLegs([]*shipment.Shipment{legA, legB}, pulid.MustNew("cus_"))
	require.Error(t, err)
	var typed *errortypes.Error
	require.True(t, errors.As(err, &typed))
	assert.Equal(t, "shipmentIds", typed.Field)
	assert.Equal(t, errortypes.ErrInvalidOperation, typed.Code)
}
