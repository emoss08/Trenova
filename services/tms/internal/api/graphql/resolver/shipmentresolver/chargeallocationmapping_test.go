package shipmentresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	shipmentdomain "github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentChargeAllocationsFromInput_NothingSentLeavesAllocationsAlone(t *testing.T) {
	t.Parallel()

	rows, err := shipmentChargeAllocationsFromInput(&gqlmodel.ShipmentInput{
		AdditionalCharges: []*gqlmodel.ShipmentAdditionalChargeInput{
			{AccessorialChargeID: pulid.MustNew("acc_").String()},
			nil,
		},
	}, resolvertest.AllocationAuthCtx())
	require.NoError(t, err)
	assert.Nil(t, rows, "nil means the repository must not touch stored allocations")
}

func TestShipmentChargeAllocationsFromInput_FlattensFreightAndChargeRows(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.AllocationAuthCtx()
	intel := pulid.MustNew("cus_")
	amd := pulid.MustNew("cus_")
	savedChargeID := pulid.MustNew("ac_")
	existingRowID := pulid.MustNew("chal_")
	sequence := 1
	version := 3

	rows, err := shipmentChargeAllocationsFromInput(&gqlmodel.ShipmentInput{
		FreightAllocations: []*gqlmodel.ChargeAllocationInput{
			{
				BillToCustomerID: intel.String(),
				Method:           shipmentdomain.ChargeAllocationMethodPercent,
				Percent:          new("60"),
			},
			{
				ID:               new(existingRowID.String()),
				BillToCustomerID: amd.String(),
				Method:           shipmentdomain.ChargeAllocationMethodPercent,
				Percent:          new("40"),
				Sequence:         &sequence,
				Version:          &version,
			},
		},
		AdditionalCharges: []*gqlmodel.ShipmentAdditionalChargeInput{
			{
				ID:                  new(savedChargeID.String()),
				AccessorialChargeID: pulid.MustNew("acc_").String(),
				Allocations: []*gqlmodel.ChargeAllocationInput{
					{
						BillToCustomerID: amd.String(),
						Method:           shipmentdomain.ChargeAllocationMethodAmount,
						Amount:           new("12.50"),
					},
				},
			},
			{
				AccessorialChargeID: pulid.MustNew("acc_").String(),
			},
			{
				AccessorialChargeID: pulid.MustNew("acc_").String(),
				Allocations: []*gqlmodel.ChargeAllocationInput{
					{
						BillToCustomerID: amd.String(),
						Method:           shipmentdomain.ChargeAllocationMethodPercent,
						Percent:          new("100"),
					},
				},
			},
		},
	}, authCtx)
	require.NoError(t, err)
	require.Len(t, rows, 4)

	freight := rows[0]
	assert.Equal(t, shipmentdomain.ChargeAllocationKindFreight, freight.ChargeKind)
	assert.Equal(t, intel, freight.BillToCustomerID)
	assert.Equal(t, authCtx.OrganizationID, freight.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, freight.BusinessUnitID)
	assert.True(t, freight.Percent.Decimal.Equal(decimal.NewFromInt(60)))
	assert.False(t, freight.Amount.Valid)
	assert.Nil(t, freight.AdditionalChargeIndex)
	assert.Nil(t, freight.AdditionalChargeID)

	second := rows[1]
	assert.Equal(t, existingRowID, second.ID)
	assert.Equal(t, int16(1), second.Sequence)
	assert.Equal(t, int64(3), second.Version)

	saved := rows[2]
	assert.Equal(t, shipmentdomain.ChargeAllocationKindAccessorial, saved.ChargeKind)
	require.NotNil(t, saved.AdditionalChargeID)
	assert.Equal(t, savedChargeID, *saved.AdditionalChargeID, "a saved charge is named by id")
	assert.Nil(t, saved.AdditionalChargeIndex)
	assert.True(t, saved.Amount.Decimal.Equal(decimal.RequireFromString("12.50")))
	assert.Equal(t, shipmentdomain.ChargeAllocationMethodAmount, saved.Method)

	unsaved := rows[3]
	assert.Equal(t, shipmentdomain.ChargeAllocationKindAccessorial, unsaved.ChargeKind)
	assert.Nil(t, unsaved.AdditionalChargeID)
	require.NotNil(t, unsaved.AdditionalChargeIndex)
	assert.Equal(t, 2, *unsaved.AdditionalChargeIndex, "an id-less charge is named by its position")
}

func TestShipmentChargeAllocationsFromInput_EmptyListsClearTheSplit(t *testing.T) {
	t.Parallel()

	rows, err := shipmentChargeAllocationsFromInput(&gqlmodel.ShipmentInput{
		FreightAllocations: []*gqlmodel.ChargeAllocationInput{},
	}, resolvertest.AllocationAuthCtx())
	require.NoError(t, err)
	require.NotNil(t, rows)
	assert.Empty(t, rows, "an empty list bills everything to the shipment's payer")
}
