package carriersettlementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newFlatAssignment() *shipment.CarrierAssignment {
	assignment := &shipment.CarrierAssignment{
		ID:            pulid.ID("casn_test"),
		CarrierID:     pulid.MustNew("car_"),
		Status:        shipment.CarrierAssignmentStatusConfirmed,
		RateMethod:    shipment.CarrierRateMethodFlat,
		BaseRate:      decimal.NewFromFloat(1500),
		FuelSurcharge: decimal.NewFromFloat(125.50),
		CurrencyCode:  "USD",
		Accessorials: []*shipment.CarrierAssignmentAccessorial{
			{Description: "Lumper", Amount: decimal.NewFromFloat(75)},
			{Description: "Detention", Amount: decimal.NewFromFloat(50.25)},
		},
	}
	assignment.SyncTotals(nil)
	return assignment
}

func TestBuildAssignmentCostEventInputsFlat(t *testing.T) {
	inputs := BuildAssignmentCostEventInputs(newFlatAssignment())

	require.Len(t, inputs, 4)
	assert.Equal(t, carriersettlement.CostEventTypeLinehaulCost, inputs[0].EventType)
	assert.Equal(t, int64(150_000), inputs[0].AmountMinor)
	assert.Equal(t, "carrier-cost:casn_test:LinehaulCost:0", inputs[0].IdempotencyKey)

	assert.Equal(t, carriersettlement.CostEventTypeFuelSurcharge, inputs[1].EventType)
	assert.Equal(t, int64(12_550), inputs[1].AmountMinor)
	assert.Equal(t, "carrier-cost:casn_test:FuelSurcharge:0", inputs[1].IdempotencyKey)

	assert.Equal(t, carriersettlement.CostEventTypeAccessorial, inputs[2].EventType)
	assert.Equal(t, int64(7_500), inputs[2].AmountMinor)
	assert.Equal(t, "Lumper", inputs[2].Description)
	assert.Equal(t, "carrier-cost:casn_test:Accessorial:0", inputs[2].IdempotencyKey)

	assert.Equal(t, carriersettlement.CostEventTypeAccessorial, inputs[3].EventType)
	assert.Equal(t, int64(5_025), inputs[3].AmountMinor)
	assert.Equal(t, "carrier-cost:casn_test:Accessorial:1", inputs[3].IdempotencyKey)
}

func TestBuildAssignmentCostEventInputsPerMile(t *testing.T) {
	distance := 425.5
	assignment := &shipment.CarrierAssignment{
		ID:           pulid.ID("casn_permile"),
		CarrierID:    pulid.MustNew("car_"),
		RateMethod:   shipment.CarrierRateMethodPerMile,
		BaseRate:     decimal.NewFromFloat(2.15),
		CurrencyCode: "USD",
	}
	assignment.SyncTotals(&distance)

	inputs := BuildAssignmentCostEventInputs(assignment)
	require.Len(t, inputs, 1)
	assert.Equal(t, carriersettlement.CostEventTypeLinehaulCost, inputs[0].EventType)
	assert.Equal(t, int64(91_482), inputs[0].AmountMinor)
	assert.Contains(t, inputs[0].Description, "PerMile")
}

func TestBuildAssignmentCostEventInputsSkipsZeroFuelSurcharge(t *testing.T) {
	assignment := newFlatAssignment()
	assignment.FuelSurcharge = decimal.Zero
	assignment.Accessorials = nil
	assignment.SyncTotals(nil)

	inputs := BuildAssignmentCostEventInputs(assignment)
	require.Len(t, inputs, 1)
	assert.Equal(t, carriersettlement.CostEventTypeLinehaulCost, inputs[0].EventType)
}

func TestBuildAssignmentCostEventInputsNilAssignment(t *testing.T) {
	assert.Nil(t, BuildAssignmentCostEventInputs(nil))
}

func TestMinorUnitsUsesBankersRounding(t *testing.T) {
	assert.Equal(t, int64(12), money.MinorUnits(decimal.NewFromFloat(0.125)))
	assert.Equal(t, int64(14), money.MinorUnits(decimal.NewFromFloat(0.135)))
	assert.Equal(t, int64(13), money.MinorUnits(decimal.NewFromFloat(0.134)))

	assignment := newFlatAssignment()
	assignment.FuelSurcharge = decimal.NewFromFloat(0.125)
	assignment.Accessorials = nil
	assignment.SyncTotals(nil)
	inputs := BuildAssignmentCostEventInputs(assignment)
	require.Len(t, inputs, 2)
	assert.Equal(t, int64(12), inputs[1].AmountMinor)
}
