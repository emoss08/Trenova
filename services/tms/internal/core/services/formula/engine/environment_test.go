package engine_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestShipment struct {
	ProNumber      string
	Weight         int64
	Pieces         int64
	TemperatureMin *int16
	TemperatureMax *int16
	Customer       *Customer
	Moves          []Move
	Commodities    []TestCommodity
}

type Customer struct {
	Name string
	Code string
}

type Move struct {
	Distance float64
	Stops    []Stop
}

type Stop struct {
	Name string
}

type TestCommodity struct {
	Weight    int64
	Pieces    int64
	Commodity *CommodityDetail
}

type CommodityDetail struct {
	HazardousMaterial *HazardousMaterial
}

type HazardousMaterial struct {
	Class string
}

func setupEnvironmentBuilder(t *testing.T) (*engine.EnvironmentBuilder, *schema.Registry) {
	t.Helper()

	registry := schema.NewRegistry()
	res := resolver.NewResolver()
	resolver.RegisterDefaultComputed(res)

	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})

	return envBuilder, registry
}

func TestEnvironmentBuilder_Build_WithoutSchema(t *testing.T) {
	builder, _ := setupEnvironmentBuilder(t)

	minTemp := int16(32)
	maxTemp := int16(40)

	entity := &TestShipment{
		ProNumber:      "PRO123",
		Weight:         5000,
		Pieces:         100,
		TemperatureMin: &minTemp,
		TemperatureMax: &maxTemp,
		Moves: []Move{
			{Distance: 100.5, Stops: []Stop{{Name: "A"}, {Name: "B"}}},
			{Distance: 200.3, Stops: []Stop{{Name: "C"}}},
		},
		Commodities: []TestCommodity{
			{Weight: 1000, Pieces: 10},
			{Weight: 2000, Pieces: 20},
		},
	}

	env, err := builder.Build(entity, "nonexistent-schema")
	require.NoError(t, err)
	require.NotNil(t, env)

	assert.InDelta(t, 300.8, env["totalDistance"], 0.1)
	assert.Equal(t, 3, env["totalStops"])
	assert.Equal(t, false, env["hasHazmat"])
	assert.Equal(t, true, env["requiresTemperatureControl"])
	assert.Equal(t, 8.0, env["temperatureDifferential"])
	assert.Equal(t, 5000.0, env["totalWeight"])
	assert.Equal(t, int64(100), env["totalPieces"])
}

func TestEnvironmentBuilder_Build_FallbackToComputed(t *testing.T) {
	builder, _ := setupEnvironmentBuilder(t)

	entity := &TestShipment{
		ProNumber: "PRO123",
		Weight:    5000,
		Customer: &Customer{
			Name: "Acme Corp",
			Code: "ACME",
		},
		Moves: []Move{
			{Distance: 100.0},
		},
	}

	env, err := builder.Build(entity, "unknown-schema")
	require.NoError(t, err)
	require.NotNil(t, env)

	assert.Equal(t, 100.0, env["totalDistance"])
}

func TestEnvironmentBuilder_Build_WithNilNested(t *testing.T) {
	builder, _ := setupEnvironmentBuilder(t)

	entity := &TestShipment{
		ProNumber: "PRO123",
		Customer:  nil,
	}

	env, err := builder.Build(entity, "unknown-schema")
	require.NoError(t, err)
	require.NotNil(t, env)

	assert.Equal(t, false, env["hasHazmat"])
}

func TestEnvironmentBuilder_BuildWithVariables(t *testing.T) {
	builder, _ := setupEnvironmentBuilder(t)

	entity := &TestShipment{
		ProNumber: "PRO123",
		Moves: []Move{
			{Distance: 100.0},
		},
	}

	variables := map[string]any{
		"baseRate":    2.5,
		"hazmatFee":   150.0,
		"customValue": "test",
	}

	env, err := builder.BuildWithVariables(entity, "nonexistent", variables)
	require.NoError(t, err)
	require.NotNil(t, env)

	assert.Equal(t, 2.5, env["baseRate"])
	assert.Equal(t, 150.0, env["hazmatFee"])
	assert.Equal(t, "test", env["customValue"])
	assert.Equal(t, 100.0, env["totalDistance"])
}

func TestEnvironmentBuilder_GetRequiredPreloads(t *testing.T) {
	builder, _ := setupEnvironmentBuilder(t)

	preloads := builder.GetRequiredPreloads("nonexistent")
	assert.Nil(t, preloads)
}

func TestEnvironmentBuilder_GetAvailableVariables(t *testing.T) {
	builder, registry := setupEnvironmentBuilder(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-variables",
		"type": "object",
		"x-formula-context": {
			"entityType": "TestShipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": []
		},
		"properties": {
			"weight": {
				"type": "number",
				"x-source": {
					"field": "Weight"
				}
			},
			"pieces": {
				"type": "number",
				"x-source": {
					"field": "Pieces"
				}
			}
		}
	}`

	err := registry.Register("test-variables", []byte(schemaJSON))
	require.NoError(t, err)

	variables := builder.GetAvailableVariables("test-variables")
	require.NotNil(t, variables)
	assert.Len(t, variables, 2)
}

func TestEnvironmentBuilder_GetAvailableVariables_SchemaNotFound(t *testing.T) {
	builder, _ := setupEnvironmentBuilder(t)

	variables := builder.GetAvailableVariables("nonexistent")
	assert.Nil(t, variables)
}

func TestEnvironmentBuilder_BuildValidationEnvironment_UsesSchemaTypes(t *testing.T) {
	t.Parallel()

	builder, registry := setupEnvironmentBuilder(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-validation-env",
		"type": "object",
		"x-formula-context": {
			"entityType": "TestShipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": []
		},
		"properties": {
			"hasHazmat": {
				"type": "boolean",
				"x-source": {
					"computed": true,
					"function": "computeHasHazmat"
				}
			},
			"customer": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"x-source": {
							"path": "Customer.Name"
						}
					},
					"code": {
						"type": "string",
						"x-source": {
							"path": "Customer.Code"
						}
					}
				}
			},
			"ratingUnit": {
				"type": "integer",
				"x-source": {
					"path": "RatingUnit"
				}
			}
		}
	}`

	err := registry.Register("test-validation-env", []byte(schemaJSON))
	require.NoError(t, err)

	env, err := builder.BuildValidationEnvironment("test-validation-env", map[string]any{
		"customer": map[string]any{"name": "Acme"},
	})
	require.NoError(t, err)

	assert.Equal(t, false, env["hasHazmat"])
	assert.Equal(t, int64(0), env["ratingUnit"])
	assert.Equal(t, map[string]any{
		"name": "Acme",
		"code": "",
	}, env["customer"])
}

func TestEnvironmentBuilder_Build_ComputedFields(t *testing.T) {
	builder, registry := setupEnvironmentBuilder(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-computed",
		"type": "object",
		"x-formula-context": {
			"entityType": "TestShipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": ["Moves"]
		},
		"properties": {
			"totalDistance": {
				"type": "number",
				"x-source": {
					"computed": true,
					"function": "computeTotalDistance"
				}
			},
			"totalStops": {
				"type": "integer",
				"x-source": {
					"computed": true,
					"function": "computeTotalStops"
				}
			}
		}
	}`

	err := registry.Register("test-computed", []byte(schemaJSON))
	require.NoError(t, err)

	entity := &TestShipment{
		Moves: []Move{
			{Distance: 100.5, Stops: []Stop{{Name: "A"}, {Name: "B"}}},
			{Distance: 200.3, Stops: []Stop{{Name: "C"}}},
		},
	}

	env, err := builder.Build(entity, "test-computed")
	require.NoError(t, err)
	require.NotNil(t, env)

	assert.InDelta(t, 300.8, env["totalDistance"], 0.1)
	assert.Equal(t, 3, env["totalStops"])
}
