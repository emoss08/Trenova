package formula_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/types/formula"
	"github.com/emoss08/trenova/internal/pkg/formula/expression"
	"github.com/emoss08/trenova/internal/pkg/formula/schema"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
	"github.com/emoss08/trenova/internal/pkg/formula/variables/builtin"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// * TestCompleteFormulaIntegration tests the entire formula system with proper variable registration
func TestCompleteFormulaIntegration(t *testing.T) {
	ctx := context.Background()

	// * Step 1: Initialize components
	varRegistry := variables.NewRegistry()
	builtin.RegisterAll(varRegistry)

	// * Register schema-based variables for shipment fields
	// These would normally be registered via the schema system
	registerShipmentVariables(varRegistry)

	resolver := schema.NewDefaultDataResolver()
	schema.RegisterShipmentComputers(resolver)

	// * Step 2: Create test shipment
	testShipment := createTestShipment()

	// * Step 3: Create variable context
	varCtx := variables.NewDefaultContext(testShipment, resolver)
	
	// Add metadata (these are dynamic values, not schema fields)
	varCtx.SetMetadata("base_rate", 1.5)
	varCtx.SetMetadata("hazmat_fee", 250.0)
	varCtx.SetMetadata("temp_control_fee", 150.0)
	varCtx.SetMetadata("distance", 750.0)
	varCtx.SetMetadata("fuel_surcharge_rate", 0.15)

	// * Step 4: Test builtin variables (these are registered by the system)
	t.Run("builtin_variables", func(t *testing.T) {
		evaluator := expression.NewEvaluator(varRegistry)
		
		// Temperature differential (computed field)
		result, err := evaluator.Evaluate(ctx, "temperature_differential", varCtx)
		require.NoError(t, err)
		assert.Equal(t, 30.0, result) // 32 - 2
		
		// Has hazmat (computed field)
		result, err = evaluator.Evaluate(ctx, "has_hazmat", varCtx)
		require.NoError(t, err)
		assert.Equal(t, 1.0, result) // true = 1
		
		// Requires temperature control (computed field)
		result, err = evaluator.Evaluate(ctx, "requires_temperature_control", varCtx)
		require.NoError(t, err)
		assert.Equal(t, 1.0, result) // true = 1
	})

	// * Step 5: Test metadata-based calculations
	t.Run("metadata_calculations", func(t *testing.T) {
		evaluator := expression.NewEvaluator(varRegistry)
		
		// These use metadata values, not schema fields
		testCases := []struct {
			name       string
			expression string
			expected   float64
		}{
			{
				name:       "basic_rate",
				expression: "base_rate * distance",
				expected:   1125.0, // 1.5 * 750
			},
			{
				name:       "conditional_hazmat",
				expression: "has_hazmat ? hazmat_fee : 0",
				expected:   250.0,
			},
			{
				name:       "conditional_temp_control",
				expression: "requires_temperature_control ? temp_control_fee : 0",
				expected:   150.0,
			},
			{
				name:       "complex_formula",
				expression: "(base_rate * distance) + (has_hazmat ? hazmat_fee : 0) + (requires_temperature_control ? temp_control_fee : 0)",
				expected:   1525.0, // 1125 + 250 + 150
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := evaluator.Evaluate(ctx, tc.expression, varCtx)
				require.NoError(t, err, "Expression: %s", tc.expression)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	// * Step 6: Test computed fields directly
	t.Run("computed_fields", func(t *testing.T) {
		// Test temperature differential computation
		fieldSource := &schema.FieldSource{
			Computed: true,
			Function: "computeTemperatureDifferential",
		}
		result, err := resolver.ResolveComputed(testShipment, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, 30.0, result) // 32 - 2 from our modified test shipment
		
		// Test has hazmat computation
		fieldSource = &schema.FieldSource{
			Computed: true,
			Function: "computeHasHazmat",
		}
		result, err = resolver.ResolveComputed(testShipment, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, true, result)
		
		// Test total stops computation
		fieldSource = &schema.FieldSource{
			Computed: true,
			Function: "computeTotalStops",
		}
		result, err = resolver.ResolveComputed(testShipment, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, 4, result) // 2 + 2 stops
	})

	// * Step 7: Test schema field resolution
	t.Run("schema_field_resolution", func(t *testing.T) {
		// Direct field access
		fieldSource := &schema.FieldSource{
			Path: "ProNumber",
		}
		result, err := resolver.ResolveField(testShipment, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, "TEST-2024-001", result)
		
		// Nested field access
		fieldSource = &schema.FieldSource{
			Path: "Weight",
		}
		result, err = resolver.ResolveField(testShipment, fieldSource)
		require.NoError(t, err)
		assert.Equal(t, int64(50000), result)
	})
}

// * registerShipmentVariables registers common shipment variables
// In production, these would be registered from the schema
func registerShipmentVariables(registry *variables.Registry) {
	// Register weight variable
	weightVar := variables.NewVariable(
		"weight",
		"Shipment weight in pounds",
		formula.ValueTypeNumber,
		variables.SourceShipment,
		func(ctx variables.VariableContext) (any, error) {
			weight, err := ctx.GetField("Weight")
			if err != nil || weight == nil {
				return 0.0, nil
			}
			if w, ok := weight.(*int64); ok && w != nil {
				return float64(*w), nil
			}
			if w, ok := weight.(int64); ok {
				return float64(w), nil
			}
			return 0.0, nil
		},
	)
	registry.Register(weightVar)
	
	// Register pieces variable
	piecesVar := variables.NewVariable(
		"pieces",
		"Number of pieces",
		formula.ValueTypeNumber,
		variables.SourceShipment,
		func(ctx variables.VariableContext) (any, error) {
			pieces, err := ctx.GetField("Pieces")
			if err != nil || pieces == nil {
				return 0.0, nil
			}
			if p, ok := pieces.(*int64); ok && p != nil {
				return float64(*p), nil
			}
			if p, ok := pieces.(int64); ok {
				return float64(p), nil
			}
			return 0.0, nil
		},
	)
	registry.Register(piecesVar)
	
	// Register computed total_stops
	totalStopsVar := variables.NewVariable(
		"total_stops",
		"Total number of stops",
		formula.ValueTypeNumber,
		variables.SourceShipment,
		func(ctx variables.VariableContext) (any, error) {
			stops, err := ctx.GetComputed("computeTotalStops")
			if err != nil {
				return 0.0, err
			}
			if s, ok := stops.(int); ok {
				return float64(s), nil
			}
			if s, ok := stops.(float64); ok {
				return s, nil
			}
			return 0.0, nil
		},
	)
	registry.Register(totalStopsVar)
	
	// Register move_count as metadata
	moveCountVar := variables.NewVariable(
		"move_count",
		"Number of moves",
		formula.ValueTypeNumber,
		variables.SourceShipment,
		func(ctx variables.VariableContext) (any, error) {
			metadata := ctx.GetMetadata()
			if count, ok := metadata["move_count"]; ok {
				if c, ok := count.(float64); ok {
					return c, nil
				}
				if c, ok := count.(int); ok {
					return float64(c), nil
				}
			}
			return 0.0, nil
		},
	)
	registry.Register(moveCountVar)
	
	// Register metadata variables
	baseRateVar := variables.NewVariable(
		"base_rate",
		"Base rate per mile",
		formula.ValueTypeNumber,
		variables.SourceCustom,
		func(ctx variables.VariableContext) (any, error) {
			metadata := ctx.GetMetadata()
			if rate, ok := metadata["base_rate"]; ok {
				if r, ok := rate.(float64); ok {
					return r, nil
				}
				if r, ok := rate.(int); ok {
					return float64(r), nil
				}
			}
			return 0.0, nil
		},
	)
	registry.Register(baseRateVar)
	
	distanceVar := variables.NewVariable(
		"distance",
		"Total distance",
		formula.ValueTypeNumber,
		variables.SourceCustom,
		func(ctx variables.VariableContext) (any, error) {
			metadata := ctx.GetMetadata()
			if dist, ok := metadata["distance"]; ok {
				if d, ok := dist.(float64); ok {
					return d, nil
				}
				if d, ok := dist.(int); ok {
					return float64(d), nil
				}
			}
			return 0.0, nil
		},
	)
	registry.Register(distanceVar)
	
	hazmatFeeVar := variables.NewVariable(
		"hazmat_fee",
		"Hazmat surcharge fee",
		formula.ValueTypeNumber,
		variables.SourceCustom,
		func(ctx variables.VariableContext) (any, error) {
			metadata := ctx.GetMetadata()
			if fee, ok := metadata["hazmat_fee"]; ok {
				if f, ok := fee.(float64); ok {
					return f, nil
				}
				if f, ok := fee.(int); ok {
					return float64(f), nil
				}
			}
			return 0.0, nil
		},
	)
	registry.Register(hazmatFeeVar)
	
	tempControlFeeVar := variables.NewVariable(
		"temp_control_fee",
		"Temperature control fee",
		formula.ValueTypeNumber,
		variables.SourceCustom,
		func(ctx variables.VariableContext) (any, error) {
			metadata := ctx.GetMetadata()
			if fee, ok := metadata["temp_control_fee"]; ok {
				if f, ok := fee.(float64); ok {
					return f, nil
				}
				if f, ok := fee.(int); ok {
					return float64(f), nil
				}
			}
			return 0.0, nil
		},
	)
	registry.Register(tempControlFeeVar)
}

// * createTestShipment creates a comprehensive test shipment
func createTestShipment() *shipment.Shipment {
	return &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ProNumber:      "TEST-2024-001",
		Status:         shipment.StatusInTransit,
		Weight:         int64Ptr(50000),
		Pieces:         int64Ptr(10),
		TemperatureMin: int16Ptr(2),  // Changed from -10 to 2 for a different test
		TemperatureMax: int16Ptr(32), // 32°F
		Commodities: []*shipment.ShipmentCommodity{
			{
				ID:          pulid.MustNew("sc_"),
				ShipmentID:  pulid.MustNew("shp_"),
				CommodityID: pulid.MustNew("com_"),
				Weight:      25000,
				Pieces:      10,
				Commodity: &commodity.Commodity{
					ID:                  pulid.MustNew("com_"),
					Name:                "Hazardous Chemical",
					HazardousMaterialID: &[]pulid.ID{pulid.MustNew("hm_")}[0],
					HazardousMaterial: &hazardousmaterial.HazardousMaterial{
						ID:           pulid.MustNew("hm_"),
						Name:         "Flammable Liquid",
						Class:        hazardousmaterial.HazardousClass3,
						PackingGroup: hazardousmaterial.PackingGroupII,
						UNNumber:     "UN1203",
					},
				},
			},
		},
		Moves: []*shipment.ShipmentMove{
			{
				ID:         pulid.MustNew("mv_"),
				ShipmentID: pulid.MustNew("shp_"),
				Sequence:   1,
				Distance:   float64Ptr(450),
				Stops: []*shipment.Stop{
					{ID: pulid.MustNew("stp_"), Type: shipment.StopTypePickup},
					{ID: pulid.MustNew("stp_"), Type: shipment.StopTypeDelivery},
				},
			},
			{
				ID:         pulid.MustNew("mv_"),
				ShipmentID: pulid.MustNew("shp_"),
				Sequence:   2,
				Distance:   float64Ptr(300),
				Stops: []*shipment.Stop{
					{ID: pulid.MustNew("stp_"), Type: shipment.StopTypePickup},
					{ID: pulid.MustNew("stp_"), Type: shipment.StopTypeDelivery},
				},
			},
		},
	}
}

// Helper functions for the complete test
func int64Ptr(v int64) *int64 {
	return &v
}

func int16Ptr(v int16) *int16 {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}