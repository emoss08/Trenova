package chargeallocation

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findStale(
	t *testing.T,
	shp *shipment.Shipment,
	allocations []*shipment.ChargeAllocation,
) []StaleAmountSplit {
	t.Helper()

	rebalance, err := RebalanceAmountSplits(t.Context(), shp, allocations)
	require.NoError(t, err)

	return rebalance.Stale
}

func convertSplits(
	t *testing.T,
	shp *shipment.Shipment,
	allocations []*shipment.ChargeAllocation,
) int {
	t.Helper()

	rebalance, err := RebalanceAmountSplits(t.Context(), shp, allocations)
	require.NoError(t, err)

	return rebalance.ConvertToPercent()
}

func TestRebalanceAmountSplits_FindsFreightAmountsNoLongerAddUp(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("3000")
	acme, peak := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	allocations := []*shipment.ChargeAllocation{
		amountAllocation(shipment.ChargeAllocationKindFreight, peak, "1500", 0),
		amountAllocation(shipment.ChargeAllocationKindFreight, acme, "1350", 1),
	}

	stale := findStale(t, shp, allocations)

	require.Len(t, stale, 1)
	assert.Equal(t, shipment.ChargeAllocationKindFreight, stale[0].Kind)
	assert.Equal(t, -1, stale[0].ChargeIndex)
	assert.True(t, stale[0].AllocatedTotal.Equal(allocDec("2850")))
	assert.True(t, stale[0].ChargeTotal.Equal(allocDec("3000")))
}

func TestRebalanceAmountSplits_FindsIgnoresPercentSplitsAndBalancedAmounts(t *testing.T) {
	t.Parallel()

	detention := flatCharge("100", 1)
	shp := allocationShipment("2850", detention)
	acme, peak := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	allocations := []*shipment.ChargeAllocation{
		amountAllocation(shipment.ChargeAllocationKindFreight, peak, "1500", 0),
		amountAllocation(shipment.ChargeAllocationKindFreight, acme, "1350", 1),
		forCharge(
			percentAllocation(shipment.ChargeAllocationKindAccessorial, peak, "60", 0),
			detention,
		),
		forCharge(
			percentAllocation(shipment.ChargeAllocationKindAccessorial, acme, "40", 1),
			detention,
		),
	}

	assert.Empty(t, findStale(t, shp, allocations))
}

func TestRebalanceAmountSplits_FindsAccessorialIsNamedByItsIndex(t *testing.T) {
	t.Parallel()

	fuel := flatCharge("50", 1)
	detention := flatCharge("200", 1)
	shp := allocationShipment("1000", fuel, detention)
	peak := pulid.MustNew("cus_")
	allocations := []*shipment.ChargeAllocation{
		forCharge(
			amountAllocation(shipment.ChargeAllocationKindAccessorial, peak, "150", 0),
			detention,
		),
	}

	stale := findStale(t, shp, allocations)

	require.Len(t, stale, 1)
	assert.Equal(t, 1, stale[0].ChargeIndex)
	assert.Equal(t, detention.ID, stale[0].AdditionalChargeID)
	assert.True(t, stale[0].ChargeTotal.Equal(allocDec("200")))
}

func TestRebalanceAmountSplits_ConvertsKeepsEachPayersProportion(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("3000")
	acme, peak := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	allocations := []*shipment.ChargeAllocation{
		amountAllocation(shipment.ChargeAllocationKindFreight, peak, "1500", 0),
		amountAllocation(shipment.ChargeAllocationKindFreight, acme, "1350", 1),
	}

	converted := convertSplits(t, shp, allocations)

	assert.Equal(t, 1, converted)
	for _, row := range allocations {
		assert.Equal(t, shipment.ChargeAllocationMethodPercent, row.Method)
		assert.False(t, row.Amount.Valid)
	}
	assert.Equal(t, "52.631579", allocations[0].Percent.Decimal.StringFixed(6))
	assert.Equal(t, "47.368421", allocations[1].Percent.Decimal.StringFixed(6))

	resolution, err := ResolveShares(t.Context(), shp, allocations)
	require.NoError(t, err)
	assert.True(t, resolution.ShareFor(peak).TotalAmount.Add(resolution.ShareFor(acme).TotalAmount).
		Equal(allocDec("3000")))
	assert.Empty(t, findStale(t, shp, allocations))
}

func TestRebalanceAmountSplits_ConvertsLeavesBalancedSplitsAlone(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("2850")
	acme, peak := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	allocations := []*shipment.ChargeAllocation{
		amountAllocation(shipment.ChargeAllocationKindFreight, peak, "1500", 0),
		amountAllocation(shipment.ChargeAllocationKindFreight, acme, "1350", 1),
	}

	assert.Zero(t, convertSplits(t, shp, allocations))
	assert.Equal(t, shipment.ChargeAllocationMethodAmount, allocations[0].Method)
}

func TestRebalanceAmountSplits_ConvertsThreeWaySplitTotalsExactlyOneHundred(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("1000")
	a, b, c := pulid.MustNew("cus_"), pulid.MustNew("cus_"), pulid.MustNew("cus_")
	allocations := []*shipment.ChargeAllocation{
		amountAllocation(shipment.ChargeAllocationKindFreight, a, "100", 0),
		amountAllocation(shipment.ChargeAllocationKindFreight, b, "100", 1),
		amountAllocation(shipment.ChargeAllocationKindFreight, c, "100", 2),
	}

	require.Equal(t, 1, convertSplits(t, shp, allocations))

	total := allocDec("0")
	for _, row := range allocations {
		total = total.Add(row.Percent.Decimal)
	}
	assert.True(t, total.Equal(allocDec("100")), "percents total %s", total)
}

func TestResolveShares_RecordsTheSplitMethodOnEachShare(t *testing.T) {
	t.Parallel()

	detention := flatCharge("100", 1)
	shp := allocationShipment("2850", detention)
	peak := pulid.MustNew("cus_")
	allocations := []*shipment.ChargeAllocation{
		amountAllocation(shipment.ChargeAllocationKindFreight, peak, "1500", 0),
		amountAllocation(shipment.ChargeAllocationKindFreight, shp.CustomerID, "1350", 1),
	}

	resolution, err := ResolveShares(t.Context(), shp, allocations)
	require.NoError(t, err)

	for _, charge := range resolution.ShareFor(peak).Charges {
		assert.Equal(t, shipment.ChargeAllocationMethodAmount, charge.Method)
	}
	for _, charge := range resolution.ShareFor(shp.CustomerID).Charges {
		if charge.Kind == shipment.ChargeAllocationKindAccessorial {
			assert.Empty(t, charge.Method, "an unsplit charge has no split method")
		}
	}
}
