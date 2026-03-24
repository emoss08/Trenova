//go:build integration

package formula_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/services/formula"
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

type IntegrationShipment struct {
	ProNumber      string
	Weight         int64
	Pieces         int64
	TemperatureMin *int16
	TemperatureMax *int16
	Customer       *IntegrationCustomer
	TractorType    *IntegrationTractorType
	Moves          []IntegrationMove
	Commodities    []IntegrationCommodity
}

type IntegrationCustomer struct {
	Name string
	Code string
}

type IntegrationTractorType struct {
	Name        string
	CostPerMile float64
}

type IntegrationMove struct {
	Distance float64
	Stops    []IntegrationStop
}

type IntegrationStop struct {
	Name string
}

type IntegrationCommodity struct {
	Weight    int64
	Pieces    int64
	Commodity *IntegrationCommodityDetail
}

type IntegrationCommodityDetail struct {
	HazardousMaterial *IntegrationHazardousMaterial
}

type IntegrationHazardousMaterial struct {
	Class string
}

type mockFormulaTemplateRepository struct{}

func (m *mockFormulaTemplateRepository) Create(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	return entity, nil
}

func (m *mockFormulaTemplateRepository) Update(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) (*formulatemplate.FormulaTemplate, error) {
	return entity, nil
}

func (m *mockFormulaTemplateRepository) GetByID(
	ctx context.Context,
	req interface{},
) (*formulatemplate.FormulaTemplate, error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepository) List(
	ctx context.Context,
	req interface{},
) (interface{}, error) {
	return nil, nil
}

func (m *mockFormulaTemplateRepository) Delete(
	ctx context.Context,
	id pulid.ID,
	tenantInfo interface{},
) error {
	return nil
}

func setupIntegrationService(t *testing.T) (*formula.Service, *engine.Engine) {
	t.Helper()

	reg := schema.NewRegistry()
	res := resolver.NewResolver()
	resolver.RegisterDefaultComputed(res)

	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: reg,
		Resolver: res,
	})

	eng := engine.NewEngine(engine.Params{
		Registry:   reg,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})

	return nil, eng
}

func TestIntegration_FlatRateBilling(t *testing.T) {
	_, eng := setupIntegrationService(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Flat Rate",
		Expression: "baseRate",
		SchemaID:   "shipment",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "baseRate",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 500.0,
			},
		},
	}

	shipment := &IntegrationShipment{
		ProNumber: "PRO123",
		Weight:    5000,
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(500.0).Equal(result.Value))
}

func TestIntegration_PerMileBilling(t *testing.T) {
	_, eng := setupIntegrationService(t)

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

	shipment := &IntegrationShipment{
		ProNumber: "PRO123",
		Moves: []IntegrationMove{
			{Distance: 100.0},
			{Distance: 150.0},
		},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(625.0).Equal(result.Value))
}

func TestIntegration_PerStopBilling(t *testing.T) {
	_, eng := setupIntegrationService(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Name:       "Per Stop",
		Expression: "baseRate + (ratePerStop * totalStops)",
		SchemaID:   "shipment",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "baseRate",
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

	shipment := &IntegrationShipment{
		ProNumber: "PRO123",
		Moves: []IntegrationMove{
			{Stops: []IntegrationStop{{Name: "A"}, {Name: "B"}}},
			{Stops: []IntegrationStop{{Name: "C"}}},
		},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(250.0).Equal(result.Value))
}

func TestIntegration_CWTBilling(t *testing.T) {
	_, eng := setupIntegrationService(t)

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

	shipment := &IntegrationShipment{
		ProNumber: "PRO123",
		Weight:    4550,
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    shipment,
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(716.63).Equal(result.Value))
}

func TestIntegration_HazmatSurcharge(t *testing.T) {
	_, eng := setupIntegrationService(t)

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
		name     string
		shipment *IntegrationShipment
		want     decimal.Decimal
	}{
		{
			name: "with hazmat",
			shipment: &IntegrationShipment{
				Commodities: []IntegrationCommodity{
					{
						Commodity: &IntegrationCommodityDetail{
							HazardousMaterial: &IntegrationHazardousMaterial{Class: "3"},
						},
					},
				},
			},
			want: decimal.NewFromFloat(150.0),
		},
		{
			name: "without hazmat",
			shipment: &IntegrationShipment{
				Commodities: []IntegrationCommodity{
					{
						Commodity: &IntegrationCommodityDetail{
							HazardousMaterial: nil,
						},
					},
				},
			},
			want: decimal.NewFromFloat(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
				Template:  template,
				Entity:    tt.shipment,
				Variables: map[string]any{},
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, tt.want.Equal(result.Value))
		})
	}
}

func TestIntegration_TemperatureControlSurcharge(t *testing.T) {
	_, eng := setupIntegrationService(t)

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
		name     string
		shipment *IntegrationShipment
		want     decimal.Decimal
	}{
		{
			name: "requires temperature control",
			shipment: &IntegrationShipment{
				TemperatureMin: &minTemp,
				TemperatureMax: &maxTemp,
			},
			want: decimal.NewFromFloat(200.0),
		},
		{
			name: "no temperature control",
			shipment: &IntegrationShipment{
				TemperatureMin: nil,
				TemperatureMax: nil,
			},
			want: decimal.NewFromFloat(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
				Template:  template,
				Entity:    tt.shipment,
				Variables: map[string]any{},
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, tt.want.Equal(result.Value))
		})
	}
}

func TestIntegration_ComplexFormula_MinimumCharge(t *testing.T) {
	_, eng := setupIntegrationService(t)

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
		name     string
		shipment *IntegrationShipment
		want     decimal.Decimal
	}{
		{
			name: "calculated exceeds minimum",
			shipment: &IntegrationShipment{
				Moves: []IntegrationMove{
					{Distance: 100.0},
				},
			},
			want: decimal.NewFromFloat(250.0),
		},
		{
			name: "minimum applies",
			shipment: &IntegrationShipment{
				Moves: []IntegrationMove{
					{Distance: 10.0},
				},
			},
			want: decimal.NewFromFloat(100.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
				Template:  template,
				Entity:    tt.shipment,
				Variables: map[string]any{},
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, tt.want.Equal(result.Value))
		})
	}
}

func TestIntegration_ComplexFormula_CombinedCharges(t *testing.T) {
	_, eng := setupIntegrationService(t)

	template := &formulatemplate.FormulaTemplate{
		ID:   pulid.MustNew("FT"),
		Name: "Combined Freight Charge",
		Expression: `
			baseRate
			+ (ratePerMile * totalDistance)
			+ (ratePerStop * totalStops)
			+ (hasHazmat ? hazmatFee : 0)
			+ (requiresTemperatureControl ? tempControlFee : 0)
		`,
		SchemaID: "shipment",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{Name: "baseRate", Type: formulatypes.VariableValueTypeNumber, DefaultValue: 50.0},
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

	shipment := &IntegrationShipment{
		ProNumber:      "PRO123",
		TemperatureMin: &minTemp,
		TemperatureMax: &maxTemp,
		Moves: []IntegrationMove{
			{
				Distance: 200.0,
				Stops:    []IntegrationStop{{Name: "A"}, {Name: "B"}},
			},
			{
				Distance: 100.0,
				Stops:    []IntegrationStop{{Name: "C"}},
			},
		},
		Commodities: []IntegrationCommodity{
			{
				Commodity: &IntegrationCommodityDetail{
					HazardousMaterial: &IntegrationHazardousMaterial{Class: "3"},
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

	expected := decimal.NewFromFloat(50.0 + (1.5 * 300.0) + (25.0 * 3) + 100.0 + 75.0)
	assert.True(t, expected.Equal(result.Value), "expected %s, got %s", expected, result.Value)
}

func TestIntegration_RuntimeVariableOverride(t *testing.T) {
	_, eng := setupIntegrationService(t)

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

	shipment := &IntegrationShipment{
		Moves: []IntegrationMove{
			{Distance: 100.0},
		},
	}

	result, err := eng.Evaluate(&formulatemplatetypes.EvaluationRequest{
		Template: template,
		Entity:   shipment,
		Variables: map[string]any{
			"ratePerMile": 3.75,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(375.0).Equal(result.Value))
}
