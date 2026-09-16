package shipment

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindStaleAmountSplits_FreightAmountsNoLongerAddUp(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("3000")
	acme, peak := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	allocations := []*ChargeAllocation{
		amountAllocation(ChargeAllocationKindFreight, peak, "1500", 0),
		amountAllocation(ChargeAllocationKindFreight, acme, "1350", 1),
	}

	stale := FindStaleAmountSplits(shp, allocations)

	require.Len(t, stale, 1)
	assert.Equal(t, ChargeAllocationKindFreight, stale[0].Kind)
	assert.Equal(t, -1, stale[0].ChargeIndex)
	assert.True(t, stale[0].AllocatedTotal.Equal(allocDec("2850")))
	assert.True(t, stale[0].ChargeTotal.Equal(allocDec("3000")))
}

func TestFindStaleAmountSplits_IgnoresPercentSplitsAndBalancedAmounts(t *testing.T) {
	t.Parallel()

	detention := flatCharge("100", 1)
	shp := allocationShipment("2850", detention)
	acme, peak := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	allocations := []*ChargeAllocation{
		amountAllocation(ChargeAllocationKindFreight, peak, "1500", 0),
		amountAllocation(ChargeAllocationKindFreight, acme, "1350", 1),
		forCharge(percentAllocation(ChargeAllocationKindAccessorial, peak, "60", 0), detention),
		forCharge(percentAllocation(ChargeAllocationKindAccessorial, acme, "40", 1), detention),
	}

	assert.Empty(t, FindStaleAmountSplits(shp, allocations))
}

func TestFindStaleAmountSplits_AccessorialIsNamedByItsIndex(t *testing.T) {
	t.Parallel()

	fuel := flatCharge("50", 1)
	detention := flatCharge("200", 1)
	shp := allocationShipment("1000", fuel, detention)
	peak := pulid.MustNew("cus_")
	allocations := []*ChargeAllocation{
		forCharge(amountAllocation(ChargeAllocationKindAccessorial, peak, "150", 0), detention),
	}

	stale := FindStaleAmountSplits(shp, allocations)

	require.Len(t, stale, 1)
	assert.Equal(t, 1, stale[0].ChargeIndex)
	assert.Equal(t, detention.ID, stale[0].AdditionalChargeID)
	assert.True(t, stale[0].ChargeTotal.Equal(allocDec("200")))
}

func TestConvertAmountSplitsToPercent_KeepsEachPayersProportion(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("3000")
	acme, peak := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	allocations := []*ChargeAllocation{
		amountAllocation(ChargeAllocationKindFreight, peak, "1500", 0),
		amountAllocation(ChargeAllocationKindFreight, acme, "1350", 1),
	}

	converted := ConvertAmountSplitsToPercent(shp, allocations)

	assert.Equal(t, 1, converted)
	for _, row := range allocations {
		assert.Equal(t, ChargeAllocationMethodPercent, row.Method)
		assert.False(t, row.Amount.Valid)
	}
	assert.Equal(t, "52.631579", allocations[0].Percent.Decimal.StringFixed(6))
	assert.Equal(t, "47.368421", allocations[1].Percent.Decimal.StringFixed(6))

	resolution, err := ResolveShares(shp, allocations)
	require.NoError(t, err)
	assert.True(t, resolution.ShareFor(peak).TotalAmount.Add(resolution.ShareFor(acme).TotalAmount).
		Equal(allocDec("3000")))
	assert.Empty(t, FindStaleAmountSplits(shp, allocations))
}

func TestConvertAmountSplitsToPercent_LeavesBalancedSplitsAlone(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("2850")
	acme, peak := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	allocations := []*ChargeAllocation{
		amountAllocation(ChargeAllocationKindFreight, peak, "1500", 0),
		amountAllocation(ChargeAllocationKindFreight, acme, "1350", 1),
	}

	assert.Zero(t, ConvertAmountSplitsToPercent(shp, allocations))
	assert.Equal(t, ChargeAllocationMethodAmount, allocations[0].Method)
}

func TestConvertAmountSplitsToPercent_ThreeWaySplitTotalsExactlyOneHundred(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("1000")
	a, b, c := pulid.MustNew("cus_"), pulid.MustNew("cus_"), pulid.MustNew("cus_")
	allocations := []*ChargeAllocation{
		amountAllocation(ChargeAllocationKindFreight, a, "100", 0),
		amountAllocation(ChargeAllocationKindFreight, b, "100", 1),
		amountAllocation(ChargeAllocationKindFreight, c, "100", 2),
	}

	require.Equal(t, 1, ConvertAmountSplitsToPercent(shp, allocations))

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
	allocations := []*ChargeAllocation{
		amountAllocation(ChargeAllocationKindFreight, peak, "1500", 0),
		amountAllocation(ChargeAllocationKindFreight, shp.CustomerID, "1350", 1),
	}

	resolution, err := ResolveShares(shp, allocations)
	require.NoError(t, err)

	for _, charge := range resolution.ShareFor(peak).Charges {
		assert.Equal(t, ChargeAllocationMethodAmount, charge.Method)
	}
	for _, charge := range resolution.ShareFor(shp.CustomerID).Charges {
		if charge.Kind == ChargeAllocationKindAccessorial {
			assert.Empty(t, charge.Method, "an unsplit charge has no split method")
		}
	}
}
