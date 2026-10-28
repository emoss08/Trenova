package chargeallocation

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveShares_SubCentPercentSplitAddsUpToTheRoundedCharge(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("0.015")
	other := pulid.MustNew("cus_")

	resolution, err := ResolveShares(t.Context(), shp, []*shipment.ChargeAllocation{
		percentAllocation(shipment.ChargeAllocationKindFreight, shp.CustomerID, "50", 0),
		percentAllocation(shipment.ChargeAllocationKindFreight, other, "50", 1),
	})
	require.NoError(t, err)

	assert.Equal(t, "0.01", resolution.ShareFor(shp.CustomerID).FreightAmount.String())
	assert.Equal(t, "0.01", resolution.ShareFor(other).FreightAmount.String())
	assert.True(t, resolution.ShareFor(shp.CustomerID).Charges[0].ChargeTotal.Equal(
		allocDec("0.015"),
	))
}

func TestResolveShares_SubCentAmountRowsMustAddUpInCents(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("0.01")
	other := pulid.MustNew("cus_")

	_, err := ResolveShares(t.Context(), shp, []*shipment.ChargeAllocation{
		amountAllocation(shipment.ChargeAllocationKindFreight, shp.CustomerID, "0.005", 0),
		amountAllocation(shipment.ChargeAllocationKindFreight, other, "0.005", 1),
	})
	require.Error(t, err)

	var typed *errortypes.Error
	require.True(t, errors.As(err, &typed))
	assert.Equal(t, "freightAllocations", typed.Field)
	assert.Equal(t, "Amount allocations for this charge must add up to {0}", typed.Message)
	assert.Equal(t, []any{"0.01"}, typed.Args)
}

func TestResolveShares_UnsavedRowsSortBeforeSavedRowsOfTheirSequence(t *testing.T) {
	t.Parallel()

	shp := allocationShipment("0.03")
	saved, first, second := pulid.MustNew("cus_"), pulid.MustNew("cus_"), pulid.MustNew("cus_")
	savedRow := percentAllocation(shipment.ChargeAllocationKindFreight, saved, "33.333334", 0)
	firstRow := percentAllocation(shipment.ChargeAllocationKindFreight, first, "33.333333", 0)
	firstRow.ID = pulid.Nil
	secondRow := percentAllocation(shipment.ChargeAllocationKindFreight, second, "33.333333", 0)
	secondRow.ID = pulid.Nil

	resolution, err := ResolveShares(t.Context(), shp, []*shipment.ChargeAllocation{
		savedRow, firstRow, secondRow,
	})
	require.NoError(t, err)

	for _, payer := range []pulid.ID{saved, first, second} {
		share := resolution.ShareFor(payer)
		require.NotNil(t, share)
		assert.Equal(t, "0.01", share.FreightAmount.String())
	}
	assert.Equal(t, savedRow.ID, resolution.ShareFor(saved).Charges[0].AllocationID)
	assert.True(t, resolution.ShareFor(first).Charges[0].AllocationID.IsNil())
}
