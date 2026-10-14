//go:build integration

package formula_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/engine"
	formulaerrors "github.com/emoss08/trenova/internal/modules/rating/app/formula/errors"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func linearFootLine(
	name string,
	weight, pieces int64,
	linearFeetPerUnit float64,
) *shipment.ShipmentCommodity {
	item := &commodity.Commodity{
		ID:                pulid.MustNew("com_"),
		Name:              name,
		LinearFeetPerUnit: &linearFeetPerUnit,
	}
	return &shipment.ShipmentCommodity{
		ID:          pulid.MustNew("sc_"),
		CommodityID: item.ID,
		Weight:      weight,
		Pieces:      pieces,
		Commodity:   item,
	}
}

func ratingMethodStop(stopType shipment.StopType, sequence int64) *shipment.Stop {
	return stopAt(
		stopType,
		shipment.StopStatusNew,
		sequence,
		"Dallas",
		1_800_000_000+sequence*3_600,
	)
}

func pickupThenTwoDeliveries(firstLeg, secondLeg float64) []*shipment.ShipmentMove {
	return []*shipment.ShipmentMove{
		loadedMove(
			0,
			firstLeg,
			ratingMethodStop(shipment.StopTypePickup, 0),
			ratingMethodStop(shipment.StopTypeDelivery, 1),
		),
		loadedMove(1, secondLeg, ratingMethodStop(shipment.StopTypeDelivery, 0)),
	}
}

func createRatingMethodShipment() *shipment.Shipment {
	weight := int64(25000)
	pieces := int64(12)
	accessorial := &accessorialcharge.AccessorialCharge{
		ID:          pulid.MustNew("acc_"),
		Code:        "LUMPER",
		Description: "Lumper Service",
		Method:      accessorialcharge.MethodFlat,
	}

	return shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO-RATING-001"
		s.Weight = &weight
		s.Pieces = &pieces
		s.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(1500.00))
		s.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromFloat(1500.00))
		s.OtherChargeAmount = decimal.NewNullDecimal(decimal.NewFromFloat(250.00))
		s.TotalChargeAmount = decimal.NewNullDecimal(decimal.NewFromFloat(1750.00))
		s.Moves = pickupThenTwoDeliveries(300.0, 150.0)
		s.Commodities = []*shipment.ShipmentCommodity{
			linearFootLine("Palletized Goods", 15000, 8, 2.5),
			linearFootLine("Crated Items", 10000, 4, 3.0),
		}
		s.AdditionalCharges = []*shipment.AdditionalCharge{
			{
				ID:                  pulid.MustNew("ac_"),
				AccessorialChargeID: accessorial.ID,
				Method:              accessorialcharge.MethodFlat,
				Amount:              decimal.NewFromFloat(250.00),
				Unit:                1,
				AccessorialCharge:   accessorial,
			},
		}
	})
}

func TestRatingMethod_FlatRate(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Flat Rate",
		SchemaID:            "shipment",
		Expression:          `baseRate`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	entity := createRatingMethodShipment()

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.InDelta(t, 1500.00, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerMile(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Per Mile",
		SchemaID:            "shipment",
		Expression:          `baseRate * totalDistance`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO-PERMILE-001"
		s.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(2.50))
		s.Moves = []*shipment.ShipmentMove{loadedMove(0, 450.0)}
	})

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 2.50 * 450.0
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerStop(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Per Stop",
		SchemaID:            "shipment",
		Expression:          `baseRate * totalStops`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO-PERSTOP-001"
		s.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(125.00))
		s.Moves = []*shipment.ShipmentMove{
			loadedMove(
				0,
				100.0,
				ratingMethodStop(shipment.StopTypePickup, 0),
				ratingMethodStop(shipment.StopTypeDelivery, 1),
				ratingMethodStop(shipment.StopTypeDelivery, 2),
			),
		}
	})

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 125.00 * 3
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerPound(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Per Pound",
		SchemaID:            "shipment",
		Expression:          `baseRate * totalWeight`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	weight := int64(25000)
	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO-PERPOUND-001"
		s.Weight = &weight
		s.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(0.08))
	})

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 0.08 * 25000.0
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerPallet(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Per Pallet",
		SchemaID:            "shipment",
		Expression:          `baseRate * totalPieces`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	pieces := int64(12)
	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO-PERPALLET-001"
		s.Pieces = &pieces
		s.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(85.00))
	})

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 85.00 * 12
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerLinearFoot(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Per Linear Foot",
		SchemaID:            "shipment",
		Expression:          `baseRate * totalLinearFeet`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO-PERLINEARFOOT-001"
		s.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(35.00))
		s.Commodities = []*shipment.ShipmentCommodity{
			linearFootLine("Palletized Goods", 0, 8, 2.5),
			linearFootLine("Crated Items", 0, 4, 3.0),
		}
	})

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expectedLinearFeet := (8 * 2.5) + (4 * 3.0)
	expected := 35.00 * expectedLinearFeet
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_FlatFee_ExistingCharges(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Flat Fee (Existing Charges)",
		SchemaID:            "shipment",
		Expression:          `freightChargeAmount + otherChargeAmount`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	entity := createRatingMethodShipment()

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 1500.00 + 250.00
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerCWT(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Per CWT (Hundredweight)",
		SchemaID:            "shipment",
		Expression:          `baseRate * ceil(totalWeight / 100)`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	weight := int64(25000)
	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO-PERCWT-001"
		s.Weight = &weight
		s.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(15.00))
	})

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 15.00 * 250
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerMileWithMinimum(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Per Mile with Minimum",
		SchemaID:   "shipment",
		Expression: `max(minimumCharge, baseRate * totalDistance)`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "minimumCharge",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 500.00,
			},
		},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO-SHORT-001"
		s.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(2.50))
		s.Moves = []*shipment.ShipmentMove{loadedMove(0, 50.0)}
	})

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	calculated := 2.50 * 50.0
	expected := max(500.00, calculated)
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_RatingUnitMultiplier(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Rating Unit Multiplier",
		SchemaID:   "shipment",
		Expression: `ratingUnit * baseRate`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "ratingUnit", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 1.0},
		},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO-RATINGUNIT-001"
		s.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(150.00))
	})

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template: template,
		Entity:   entity,
		Variables: map[string]any{
			"ratingUnit": 5.0,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 5.0 * 150.00
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_CombinedRateWithAccessorials(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:       pulid.MustNew("FT"),
		Name:     "Combined Rate with Accessorials",
		SchemaID: "shipment",
		Expression: `
			round(
				max(
					minimumCharge,
					baseRate * totalDistance
				) * (1 + fuelSurchargePercent / 100) +
				(totalStops > 2 ? (totalStops - 2) * additionalStopFee : 0),
				2
			)
		`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "minimumCharge",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 300.00,
			},
			{
				Name:         "fuelSurchargePercent",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 18.0,
			},
			{
				Name:         "additionalStopFee",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 50.00,
			},
		},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO-COMBINED-001"
		s.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(2.25))
		s.Moves = pickupThenTwoDeliveries(300.0, 150.0)
	})

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	linehaul := max(300.00, 2.25*450.0)
	withFuel := linehaul * 1.18
	stopFee := (3 - 2) * 50.0
	expected := withFuel + stopFee
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_UseCurrentTotalCharge(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Use Current Total Charge",
		SchemaID:            "shipment",
		Expression:          `currentTotalCharge`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	entity := createRatingMethodShipment()

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.InDelta(t, 1750.00, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_NilChargeAmounts(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Flat Fee with Nil Amounts",
		SchemaID:            "shipment",
		Expression:          `freightChargeAmount + otherChargeAmount`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	var entity *shipment.Shipment

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.Error(t, err)
	assert.Nil(t, result)
	var unresolved *formulaerrors.UnresolvedFieldError
	require.ErrorAs(t, err, &unresolved)
	assert.Equal(t, []string{"freightChargeAmount", "otherChargeAmount"}, unresolved.Fields)
	require.Error(t, unresolved.Causes["freightChargeAmount"])
	require.Error(t, unresolved.Causes["otherChargeAmount"])
	assert.Contains(t, err.Error(), "freightChargeAmount")
	assert.Contains(t, err.Error(), "otherChargeAmount")
}

func TestRatingMethod_NonFiniteResult(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Rate Divided by Unit Count",
		SchemaID:   "shipment",
		Expression: `numerator / denominator`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "numerator", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 1.0},
			{Name: "denominator", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 1.0},
		},
	}

	tests := []struct {
		name      string
		variables map[string]any
	}{
		{
			name:      "positive infinity errors",
			variables: map[string]any{"numerator": 1500.0, "denominator": 0.0},
		},
		{
			name:      "negative infinity errors",
			variables: map[string]any{"numerator": -1500.0, "denominator": 0.0},
		},
		{
			name:      "NaN errors",
			variables: map[string]any{"numerator": 0.0, "denominator": 0.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
				Template:  template,
				Entity:    createRatingMethodShipment(),
				Variables: tt.variables,
			})

			require.Error(t, err)
			require.ErrorIs(t, err, engine.ErrNonFiniteResult)
			assert.Nil(t, result)
		})
	}
}

func TestRatingMethod_ZeroLinearFeet(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Per Linear Foot (No Commodities)",
		SchemaID:   "shipment",
		Expression: `max(minimumCharge, ratePerLinearFoot * totalLinearFeet)`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "minimumCharge",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 200.00,
			},
			{
				Name:         "ratePerLinearFoot",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 35.00,
			},
		},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO-EMPTY-001"
		s.Moves = []*shipment.ShipmentMove{}
		s.Commodities = []*shipment.ShipmentCommodity{}
	})

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.InDelta(t, 200.00, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_InvalidNullDecimalHandling(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Flat Fee with Invalid Decimal",
		SchemaID:            "shipment",
		Expression:          `freightChargeAmount + otherChargeAmount`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO-INVALID-001"
		s.FreightChargeAmount = decimal.NullDecimal{Valid: false}
		s.OtherChargeAmount = decimal.NullDecimal{Valid: false}
	})

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.InDelta(t, 0.0, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_MixedValidInvalidCharges(t *testing.T) {
	eng := newTestEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Mixed Valid/Invalid Charges",
		SchemaID:            "shipment",
		Expression:          `freightChargeAmount + otherChargeAmount`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO-MIXED-001"
		s.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromFloat(500.00))
		s.OtherChargeAmount = decimal.NullDecimal{Valid: false}
	})

	result, err := eng.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.InDelta(t, 500.00, result.Value.InexactFloat64(), 0.01)
}
