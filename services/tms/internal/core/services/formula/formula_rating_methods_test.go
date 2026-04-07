//go:build integration

package formula_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type RatingMethodShipment struct {
	ID                  string
	ProNumber           string
	Weight              *int64
	Pieces              *int64
	BaseRate            *decimal.NullDecimal
	FreightChargeAmount *decimal.NullDecimal
	OtherChargeAmount   *decimal.NullDecimal
	TotalChargeAmount   *decimal.NullDecimal
	Moves               []*RatingMethodMove
	Commodities         []*RatingMethodShipmentCommodity
}

type RatingMethodMove struct {
	ID       string
	Distance *float64
	Stops    []*RatingMethodStop
}

type RatingMethodStop struct {
	ID   string
	Type string
}

type RatingMethodShipmentCommodity struct {
	ID        string
	Weight    int64
	Pieces    int64
	Commodity *RatingMethodCommodity
}

type RatingMethodCommodity struct {
	ID                string
	Name              string
	LinearFeetPerUnit float64
}

func setupRatingMethodEngine(t *testing.T) *engine.Engine {
	t.Helper()

	reg := schema.NewRegistry()
	res := resolver.NewResolver()
	resolver.RegisterDefaultComputed(res)

	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: reg,
		Resolver: res,
	})

	return engine.NewEngine(engine.Params{
		Registry:   reg,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})
}

func createRatingMethodShipment() *RatingMethodShipment {
	weight := int64(25000)
	pieces := int64(12)
	distance1 := 300.0
	distance2 := 150.0

	baseRate := decimal.NewNullDecimal(decimal.NewFromFloat(1500.00))
	freightCharge := decimal.NewNullDecimal(decimal.NewFromFloat(1500.00))
	otherCharge := decimal.NewNullDecimal(decimal.NewFromFloat(250.00))
	totalCharge := decimal.NewNullDecimal(decimal.NewFromFloat(1750.00))

	return &RatingMethodShipment{
		ID:                  "SP_RATING",
		ProNumber:           "PRO-RATING-001",
		Weight:              &weight,
		Pieces:              &pieces,
		BaseRate:            &baseRate,
		FreightChargeAmount: &freightCharge,
		OtherChargeAmount:   &otherCharge,
		TotalChargeAmount:   &totalCharge,
		Moves: []*RatingMethodMove{
			{
				ID:       "SM_001",
				Distance: &distance1,
				Stops: []*RatingMethodStop{
					{ID: "STP_001", Type: "Pickup"},
					{ID: "STP_002", Type: "Delivery"},
				},
			},
			{
				ID:       "SM_002",
				Distance: &distance2,
				Stops: []*RatingMethodStop{
					{ID: "STP_003", Type: "Delivery"},
				},
			},
		},
		Commodities: []*RatingMethodShipmentCommodity{
			{
				ID:     "SC_001",
				Weight: 15000,
				Pieces: 8,
				Commodity: &RatingMethodCommodity{
					ID:                "COM_001",
					Name:              "Palletized Goods",
					LinearFeetPerUnit: 2.5,
				},
			},
			{
				ID:     "SC_002",
				Weight: 10000,
				Pieces: 4,
				Commodity: &RatingMethodCommodity{
					ID:                "COM_002",
					Name:              "Crated Items",
					LinearFeetPerUnit: 3.0,
				},
			},
		},
	}
}

func TestRatingMethod_FlatRate(t *testing.T) {
	eng := setupRatingMethodEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Flat Rate",
		SchemaID:            "shipment",
		Expression:          `baseRate`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	shipment := createRatingMethodShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.InDelta(t, 1500.00, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerMile(t *testing.T) {
	eng := setupRatingMethodEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Per Mile",
		SchemaID:            "shipment",
		Expression:          `baseRate * totalDistance`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	freightRate := decimal.NewNullDecimal(decimal.NewFromFloat(2.50))
	distance := 450.0
	shipment := &RatingMethodShipment{
		ID:        "SP_PERMILE",
		ProNumber: "PRO-PERMILE-001",
		BaseRate:  &freightRate,
		Moves: []*RatingMethodMove{
			{ID: "SM_001", Distance: &distance, Stops: []*RatingMethodStop{}},
		},
		Commodities: []*RatingMethodShipmentCommodity{},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 2.50 * 450.0
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerStop(t *testing.T) {
	eng := setupRatingMethodEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Per Stop",
		SchemaID:            "shipment",
		Expression:          `baseRate * totalStops`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	freightRate := decimal.NewNullDecimal(decimal.NewFromFloat(125.00))
	distance := 100.0
	shipment := &RatingMethodShipment{
		ID:        "SP_PERSTOP",
		ProNumber: "PRO-PERSTOP-001",
		BaseRate:  &freightRate,
		Moves: []*RatingMethodMove{
			{
				ID: "SM_001", Distance: &distance,
				Stops: []*RatingMethodStop{
					{ID: "STP_001", Type: "Pickup"},
					{ID: "STP_002", Type: "Delivery"},
					{ID: "STP_003", Type: "Delivery"},
				},
			},
		},
		Commodities: []*RatingMethodShipmentCommodity{},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 125.00 * 3
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerPound(t *testing.T) {
	eng := setupRatingMethodEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Per Pound",
		SchemaID:            "shipment",
		Expression:          `baseRate * totalWeight`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	freightRate := decimal.NewNullDecimal(decimal.NewFromFloat(0.08))
	weight := int64(25000)
	shipment := &RatingMethodShipment{
		ID:        "SP_PERPOUND",
		ProNumber: "PRO-PERPOUND-001",
		Weight:    &weight,
		BaseRate:  &freightRate,
		Moves:               []*RatingMethodMove{},
		Commodities:         []*RatingMethodShipmentCommodity{},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 0.08 * 25000.0
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerPallet(t *testing.T) {
	eng := setupRatingMethodEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Per Pallet",
		SchemaID:            "shipment",
		Expression:          `baseRate * totalPieces`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	freightRate := decimal.NewNullDecimal(decimal.NewFromFloat(85.00))
	pieces := int64(12)
	shipment := &RatingMethodShipment{
		ID:        "SP_PERPALLET",
		ProNumber: "PRO-PERPALLET-001",
		Pieces:    &pieces,
		BaseRate:  &freightRate,
		Moves:               []*RatingMethodMove{},
		Commodities:         []*RatingMethodShipmentCommodity{},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 85.00 * 12
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerLinearFoot(t *testing.T) {
	eng := setupRatingMethodEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Per Linear Foot",
		SchemaID:            "shipment",
		Expression:          `baseRate * totalLinearFeet`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	freightRate := decimal.NewNullDecimal(decimal.NewFromFloat(35.00))
	shipment := &RatingMethodShipment{
		ID:        "SP_PERLINEARFOOT",
		ProNumber: "PRO-PERLINEARFOOT-001",
		BaseRate:  &freightRate,
		Moves:               []*RatingMethodMove{},
		Commodities: []*RatingMethodShipmentCommodity{
			{
				ID:     "SC_001",
				Pieces: 8,
				Commodity: &RatingMethodCommodity{
					ID:                "COM_001",
					Name:              "Palletized Goods",
					LinearFeetPerUnit: 2.5,
				},
			},
			{
				ID:     "SC_002",
				Pieces: 4,
				Commodity: &RatingMethodCommodity{
					ID:                "COM_002",
					Name:              "Crated Items",
					LinearFeetPerUnit: 3.0,
				},
			},
		},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expectedLinearFeet := (8 * 2.5) + (4 * 3.0)
	expected := 35.00 * expectedLinearFeet
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_FlatFee_ExistingCharges(t *testing.T) {
	eng := setupRatingMethodEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Flat Fee (Existing Charges)",
		SchemaID:            "shipment",
		Expression:          `freightChargeAmount + otherChargeAmount`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	shipment := createRatingMethodShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 1500.00 + 250.00
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerCWT(t *testing.T) {
	eng := setupRatingMethodEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Per CWT (Hundredweight)",
		SchemaID:            "shipment",
		Expression:          `baseRate * ceil(totalWeight / 100)`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	freightRate := decimal.NewNullDecimal(decimal.NewFromFloat(15.00))
	weight := int64(25000)
	shipment := &RatingMethodShipment{
		ID:        "SP_PERCWT",
		ProNumber: "PRO-PERCWT-001",
		Weight:    &weight,
		BaseRate:  &freightRate,
		Moves:               []*RatingMethodMove{},
		Commodities:         []*RatingMethodShipmentCommodity{},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := 15.00 * 250
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_PerMileWithMinimum(t *testing.T) {
	eng := setupRatingMethodEngine(t)

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

	freightRate := decimal.NewNullDecimal(decimal.NewFromFloat(2.50))
	distance := 50.0
	shipment := &RatingMethodShipment{
		ID:        "SP_SHORT",
		ProNumber: "PRO-SHORT-001",
		BaseRate:  &freightRate,
		Moves: []*RatingMethodMove{
			{
				ID:       "SM_001",
				Distance: &distance,
				Stops:    []*RatingMethodStop{},
			},
		},
		Commodities: []*RatingMethodShipmentCommodity{},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	calculated := 2.50 * 50.0
	expected := max(500.00, calculated)
	assert.InDelta(t, expected, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_RatingUnitMultiplier(t *testing.T) {
	eng := setupRatingMethodEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Rating Unit Multiplier",
		SchemaID:   "shipment",
		Expression: `ratingUnit * baseRate`,
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "ratingUnit", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 1.0},
		},
	}

	freightRate := decimal.NewNullDecimal(decimal.NewFromFloat(150.00))
	shipment := &RatingMethodShipment{
		ID:        "SP_RATINGUNIT",
		ProNumber: "PRO-RATINGUNIT-001",
		BaseRate:  &freightRate,
		Moves:               []*RatingMethodMove{},
		Commodities:         []*RatingMethodShipmentCommodity{},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template: template,
		Entity:   shipment,
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
	eng := setupRatingMethodEngine(t)

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

	freightRate := decimal.NewNullDecimal(decimal.NewFromFloat(2.25))
	distance1 := 300.0
	distance2 := 150.0
	shipment := &RatingMethodShipment{
		ID:        "SP_COMBINED",
		ProNumber: "PRO-COMBINED-001",
		BaseRate:  &freightRate,
		Moves: []*RatingMethodMove{
			{
				ID: "SM_001", Distance: &distance1,
				Stops: []*RatingMethodStop{
					{ID: "STP_001", Type: "Pickup"},
					{ID: "STP_002", Type: "Delivery"},
				},
			},
			{
				ID: "SM_002", Distance: &distance2,
				Stops: []*RatingMethodStop{
					{ID: "STP_003", Type: "Delivery"},
				},
			},
		},
		Commodities: []*RatingMethodShipmentCommodity{},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
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
	eng := setupRatingMethodEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Use Current Total Charge",
		SchemaID:            "shipment",
		Expression:          `currentTotalCharge`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	shipment := createRatingMethodShipment()

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.InDelta(t, 1750.00, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_NilChargeAmounts(t *testing.T) {
	eng := setupRatingMethodEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Flat Fee with Nil Amounts",
		SchemaID:            "shipment",
		Expression:          `freightChargeAmount + otherChargeAmount`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	shipment := &RatingMethodShipment{
		ID:                  "SP_NIL",
		ProNumber:           "PRO-NIL-001",
		FreightChargeAmount: nil,
		OtherChargeAmount:   nil,
		Moves:               []*RatingMethodMove{},
		Commodities:         []*RatingMethodShipmentCommodity{},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.InDelta(t, 0.0, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_ZeroLinearFeet(t *testing.T) {
	eng := setupRatingMethodEngine(t)

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

	shipment := &RatingMethodShipment{
		ID:          "SP_EMPTY",
		ProNumber:   "PRO-EMPTY-001",
		Moves:       []*RatingMethodMove{},
		Commodities: []*RatingMethodShipmentCommodity{},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.InDelta(t, 200.00, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_InvalidNullDecimalHandling(t *testing.T) {
	eng := setupRatingMethodEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Flat Fee with Invalid Decimal",
		SchemaID:            "shipment",
		Expression:          `freightChargeAmount + otherChargeAmount`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	invalidDecimal := decimal.NullDecimal{Valid: false}
	shipment := &RatingMethodShipment{
		ID:                  "SP_INVALID",
		ProNumber:           "PRO-INVALID-001",
		FreightChargeAmount: &invalidDecimal,
		OtherChargeAmount:   &invalidDecimal,
		Moves:               []*RatingMethodMove{},
		Commodities:         []*RatingMethodShipmentCommodity{},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.InDelta(t, 0.0, result.Value.InexactFloat64(), 0.01)
}

func TestRatingMethod_MixedValidInvalidCharges(t *testing.T) {
	eng := setupRatingMethodEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:                  pulid.MustNew("FT"),
		Name:                "Mixed Valid/Invalid Charges",
		SchemaID:            "shipment",
		Expression:          `freightChargeAmount + otherChargeAmount`,
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}

	validCharge := decimal.NewNullDecimal(decimal.NewFromFloat(500.00))
	invalidCharge := decimal.NullDecimal{Valid: false}
	shipment := &RatingMethodShipment{
		ID:                  "SP_MIXED",
		ProNumber:           "PRO-MIXED-001",
		FreightChargeAmount: &validCharge,
		OtherChargeAmount:   &invalidCharge,
		Moves:               []*RatingMethodMove{},
		Commodities:         []*RatingMethodShipmentCommodity{},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.InDelta(t, 500.00, result.Value.InexactFloat64(), 0.01)
}
