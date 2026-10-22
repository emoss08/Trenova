//go:build integration

package formula_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadedMove(
	sequence int64,
	distance float64,
	stops ...*shipment.Stop,
) *shipment.ShipmentMove {
	return &shipment.ShipmentMove{
		ID:       pulid.MustNew("sm_"),
		Status:   shipment.MoveStatusNew,
		Loaded:   true,
		Sequence: sequence,
		Distance: &distance,
		Stops:    stops,
	}
}

func unmeasuredMove(sequence int64, stops ...*shipment.Stop) *shipment.ShipmentMove {
	move := loadedMove(sequence, 0, stops...)
	move.Distance = nil
	return move
}

func hazmatLine(class hazardousmaterial.HazardousClass) *shipment.ShipmentCommodity {
	material := &hazardousmaterial.HazardousMaterial{
		ID:    pulid.MustNew("hm_"),
		Class: class,
	}
	item := &commodity.Commodity{
		ID:                  pulid.MustNew("com_"),
		HazardousMaterialID: material.ID,
		HazardousMaterial:   material,
	}
	return &shipment.ShipmentCommodity{
		ID:          pulid.MustNew("sc_"),
		CommodityID: item.ID,
		Commodity:   item,
	}
}

func nonHazmatLine() *shipment.ShipmentCommodity {
	item := &commodity.Commodity{ID: pulid.MustNew("com_")}
	return &shipment.ShipmentCommodity{
		ID:          pulid.MustNew("sc_"),
		CommodityID: item.ID,
		Commodity:   item,
	}
}

func shipmentWith(configure func(*shipment.Shipment)) *shipment.Shipment {
	entity := ratedShipment()
	entity.ID = pulid.MustNew("shp_")
	configure(entity)
	return entity
}

func TestIntegration_FlatRateBilling(t *testing.T) {
	eng := newRawRating(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Flat Rate",
		Expression: "flatRate",
		SchemaID:   "shipment",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "flatRate",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 500.0,
			},
		},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		weight := int64(5000)
		s.ProNumber = "PRO123"
		s.Weight = &weight
	})

	result, err := eng.Evaluate(t.Context(), &evaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(500.0).Equal(result.Value))
}

func TestIntegration_PerMileBilling(t *testing.T) {
	eng := newRawRating(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Per Mile",
		Expression: "ratePerMile * totalDistance",
		SchemaID:   "shipment",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "ratePerMile",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 2.50,
			},
		},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO123"
		s.Moves = []*shipment.ShipmentMove{loadedMove(0, 100.0), loadedMove(1, 150.0)}
	})

	result, err := eng.Evaluate(t.Context(), &evaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(625.0).Equal(result.Value))
}

func TestIntegration_PerStopBilling(t *testing.T) {
	eng := newRawRating(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Per Stop",
		Expression: "startingRate + (ratePerStop * totalStops)",
		SchemaID:   "shipment",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "startingRate",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 100.0,
			},
			{
				Name:         "ratePerStop",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 50.0,
			},
		},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO123"
		s.Moves = []*shipment.ShipmentMove{
			unmeasuredMove(
				0,
				stopAt(shipment.StopTypePickup, shipment.StopStatusNew, 0, "A", 1_800_000_000),
				stopAt(shipment.StopTypeDelivery, shipment.StopStatusNew, 1, "B", 1_800_010_000),
			),
			unmeasuredMove(
				1,
				stopAt(shipment.StopTypeDelivery, shipment.StopStatusNew, 0, "C", 1_800_020_000),
			),
		}
	})

	result, err := eng.Evaluate(t.Context(), &evaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(250.0).Equal(result.Value))
}

func TestIntegration_CWTBilling(t *testing.T) {
	eng := newRawRating(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Per CWT",
		Expression: "round(ratePerCWT * (totalWeight / 100), 2)",
		SchemaID:   "shipment",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "ratePerCWT",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 15.75,
			},
		},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		weight := int64(4550)
		s.ProNumber = "PRO123"
		s.Weight = &weight
	})

	result, err := eng.Evaluate(t.Context(), &evaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(716.63).Equal(result.Value))
}

func TestIntegration_HazmatSurcharge(t *testing.T) {
	eng := newRawRating(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Hazmat Surcharge",
		Expression: "hasHazmat ? hazmatFee : 0",
		SchemaID:   "shipment",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "hazmatFee",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 150.0,
			},
		},
	}

	tests := []struct {
		name   string
		entity *shipment.Shipment
		want   decimal.Decimal
	}{
		{
			name: "with hazmat",
			entity: shipmentWith(func(s *shipment.Shipment) {
				s.Commodities = []*shipment.ShipmentCommodity{
					hazmatLine(hazardousmaterial.HazardousClass3),
				}
			}),
			want: decimal.NewFromFloat(150.0),
		},
		{
			name: "without hazmat",
			entity: shipmentWith(func(s *shipment.Shipment) {
				s.Commodities = []*shipment.ShipmentCommodity{nonHazmatLine()}
			}),
			want: decimal.NewFromFloat(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eng.Evaluate(t.Context(), &evaluationRequest{
				Template:  template,
				Entity:    tt.entity,
				Variables: map[string]any{},
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, tt.want.Equal(result.Value))
		})
	}
}

func TestIntegration_TemperatureControlSurcharge(t *testing.T) {
	eng := newRawRating(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Temperature Control Surcharge",
		Expression: "requiresTemperatureControl ? tempControlFee : 0",
		SchemaID:   "shipment",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "tempControlFee",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 200.0,
			},
		},
	}

	minTemp := int16(32)
	maxTemp := int16(40)

	tests := []struct {
		name   string
		entity *shipment.Shipment
		want   decimal.Decimal
	}{
		{
			name: "requires temperature control",
			entity: shipmentWith(func(s *shipment.Shipment) {
				s.TemperatureMin = &minTemp
				s.TemperatureMax = &maxTemp
			}),
			want: decimal.NewFromFloat(200.0),
		},
		{
			name: "no temperature control",
			entity: shipmentWith(func(s *shipment.Shipment) {
				s.TemperatureMin = nil
				s.TemperatureMax = nil
			}),
			want: decimal.NewFromFloat(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eng.Evaluate(t.Context(), &evaluationRequest{
				Template:  template,
				Entity:    tt.entity,
				Variables: map[string]any{},
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, tt.want.Equal(result.Value))
		})
	}
}

func TestIntegration_ComplexFormula_MinimumCharge(t *testing.T) {
	eng := newRawRating(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Per Mile with Minimum",
		Expression: "max(minimumCharge, ratePerMile * totalDistance)",
		SchemaID:   "shipment",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "minimumCharge",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 100.0,
			},
			{
				Name:         "ratePerMile",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 2.50,
			},
		},
	}

	tests := []struct {
		name   string
		entity *shipment.Shipment
		want   decimal.Decimal
	}{
		{
			name: "calculated exceeds minimum",
			entity: shipmentWith(func(s *shipment.Shipment) {
				s.Moves = []*shipment.ShipmentMove{loadedMove(0, 100.0)}
			}),
			want: decimal.NewFromFloat(250.0),
		},
		{
			name: "minimum applies",
			entity: shipmentWith(func(s *shipment.Shipment) {
				s.Moves = []*shipment.ShipmentMove{loadedMove(0, 10.0)}
			}),
			want: decimal.NewFromFloat(100.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eng.Evaluate(t.Context(), &evaluationRequest{
				Template:  template,
				Entity:    tt.entity,
				Variables: map[string]any{},
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, tt.want.Equal(result.Value))
		})
	}
}

func TestIntegration_ComplexFormula_CombinedCharges(t *testing.T) {
	eng := newRawRating(t)

	template := &formulatemplate.FormulaTemplate{
		ID:   pulid.MustNew("FT"),
		Name: "Combined Freight Charge",
		Expression: `
			startingRate
			+ (ratePerMile * totalDistance)
			+ (ratePerStop * totalStops)
			+ (hasHazmat ? hazmatFee : 0)
			+ (requiresTemperatureControl ? tempControlFee : 0)
		`,
		SchemaID: "shipment",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "startingRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 50.0},
			{Name: "ratePerMile", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 1.5},
			{Name: "ratePerStop", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 25.0},
			{Name: "hazmatFee", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 100.0},
			{
				Name:         "tempControlFee",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 75.0,
			},
		},
	}

	minTemp := int16(32)
	maxTemp := int16(40)

	entity := shipmentWith(func(s *shipment.Shipment) {
		s.ProNumber = "PRO123"
		s.TemperatureMin = &minTemp
		s.TemperatureMax = &maxTemp
		s.Moves = []*shipment.ShipmentMove{
			loadedMove(
				0,
				200.0,
				stopAt(shipment.StopTypePickup, shipment.StopStatusNew, 0, "A", 1_800_000_000),
				stopAt(shipment.StopTypeDelivery, shipment.StopStatusNew, 1, "B", 1_800_010_000),
			),
			loadedMove(
				1,
				100.0,
				stopAt(shipment.StopTypeDelivery, shipment.StopStatusNew, 0, "C", 1_800_020_000),
			),
		}
		s.Commodities = []*shipment.ShipmentCommodity{
			hazmatLine(hazardousmaterial.HazardousClass3),
		}
	})

	result, err := eng.Evaluate(t.Context(), &evaluationRequest{
		Template:  template,
		Entity:    entity,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	expected := decimal.NewFromFloat(50.0 + (1.5 * 300.0) + (25.0 * 3) + 100.0 + 75.0)
	assert.True(t, expected.Equal(result.Value), "expected %s, got %s", expected, result.Value)
}

func TestIntegration_RuntimeVariableOverride(t *testing.T) {
	eng := newRawRating(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Per Mile",
		Expression: "ratePerMile * totalDistance",
		SchemaID:   "shipment",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "ratePerMile",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 2.50,
			},
		},
	}

	entity := shipmentWith(func(s *shipment.Shipment) {
		s.Moves = []*shipment.ShipmentMove{loadedMove(0, 100.0)}
	})

	result, err := eng.Evaluate(t.Context(), &evaluationRequest{
		Template: template,
		Entity:   entity,
		Variables: map[string]any{
			"ratePerMile": 3.75,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(375.0).Equal(result.Value))
}
