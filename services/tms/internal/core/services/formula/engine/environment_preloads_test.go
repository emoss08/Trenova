package engine_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRequiredPreloads_WithSchemaPreloads(t *testing.T) {
	t.Parallel()

	builder, registry := setupEnvironmentBuilder(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-preloads",
		"type": "object",
		"x-formula-context": {
			"entityType": "Shipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": ["Moves", "Moves.Stops", "Commodities"]
		},
		"properties": {
			"weight": {
				"type": "number",
				"x-source": {
					"field": "Weight"
				}
			}
		}
	}`

	err := registry.Register("test-preloads", []byte(schemaJSON))
	require.NoError(t, err)

	preloads := builder.GetRequiredPreloads("test-preloads")
	require.NotNil(t, preloads)
	assert.Contains(t, preloads, "Moves")
	assert.Contains(t, preloads, "Moves.Stops")
	assert.Contains(t, preloads, "Commodities")
}

func TestGetRequiredPreloads_WithFieldPreloads(t *testing.T) {
	t.Parallel()

	builder, registry := setupEnvironmentBuilder(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-field-preloads",
		"type": "object",
		"x-formula-context": {
			"entityType": "Shipment"
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
					"function": "computeTotalDistance",
					"preload": ["Moves"]
				}
			},
			"hasHazmat": {
				"type": "boolean",
				"x-source": {
					"computed": true,
					"function": "computeHasHazmat",
					"preload": ["Commodities", "Commodities.Commodity", "Commodities.Commodity.HazardousMaterial"]
				}
			}
		}
	}`

	err := registry.Register("test-field-preloads", []byte(schemaJSON))
	require.NoError(t, err)

	preloads := builder.GetRequiredPreloads("test-field-preloads")
	require.NotNil(t, preloads)
	assert.Contains(t, preloads, "Moves")
	assert.Contains(t, preloads, "Commodities")
	assert.Contains(t, preloads, "Commodities.Commodity")
	assert.Contains(t, preloads, "Commodities.Commodity.HazardousMaterial")
}

func TestGetRequiredPreloads_DeduplicatesPreloads(t *testing.T) {
	t.Parallel()

	builder, registry := setupEnvironmentBuilder(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-dedup-preloads",
		"type": "object",
		"x-formula-context": {
			"entityType": "Shipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": ["Moves", "Commodities"]
		},
		"properties": {
			"totalDistance": {
				"type": "number",
				"x-source": {
					"computed": true,
					"function": "computeTotalDistance",
					"preload": ["Moves"]
				}
			},
			"totalWeight": {
				"type": "number",
				"x-source": {
					"computed": true,
					"function": "computeTotalWeight",
					"preload": ["Commodities"]
				}
			}
		}
	}`

	err := registry.Register("test-dedup-preloads", []byte(schemaJSON))
	require.NoError(t, err)

	preloads := builder.GetRequiredPreloads("test-dedup-preloads")
	require.NotNil(t, preloads)

	countMoves := 0
	countCommodities := 0
	for _, p := range preloads {
		if p == "Moves" {
			countMoves++
		}
		if p == "Commodities" {
			countCommodities++
		}
	}
	assert.Equal(t, 1, countMoves)
	assert.Equal(t, 1, countCommodities)
}

func TestGetRequiredPreloads_EmptyPreloads(t *testing.T) {
	t.Parallel()

	builder, registry := setupEnvironmentBuilder(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-empty-preloads",
		"type": "object",
		"x-formula-context": {
			"entityType": "Shipment"
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
			}
		}
	}`

	err := registry.Register("test-empty-preloads", []byte(schemaJSON))
	require.NoError(t, err)

	preloads := builder.GetRequiredPreloads("test-empty-preloads")
	assert.Empty(t, preloads)
}

func TestGetRequiredPreloads_SchemaNotFound(t *testing.T) {
	t.Parallel()

	builder, _ := setupEnvironmentBuilder(t)

	preloads := builder.GetRequiredPreloads("nonexistent-schema-id")
	assert.Nil(t, preloads)
}

func TestGetRequiredPreloads_MultipleFieldPreloads(t *testing.T) {
	t.Parallel()

	builder, registry := setupEnvironmentBuilder(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-multi-field-preloads",
		"type": "object",
		"x-formula-context": {
			"entityType": "Shipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": []
		},
		"properties": {
			"totalDistance": {
				"type": "number",
				"x-source": {
					"computed": true,
					"function": "computeTotalDistance",
					"preload": ["Moves"]
				}
			},
			"totalStops": {
				"type": "integer",
				"x-source": {
					"computed": true,
					"function": "computeTotalStops",
					"preload": ["Moves", "Moves.Stops"]
				}
			},
			"linearFeet": {
				"type": "number",
				"x-source": {
					"computed": true,
					"function": "computeTotalLinearFeet",
					"preload": ["Commodities", "Commodities.Commodity"]
				}
			}
		}
	}`

	err := registry.Register("test-multi-field-preloads", []byte(schemaJSON))
	require.NoError(t, err)

	preloads := builder.GetRequiredPreloads("test-multi-field-preloads")
	require.NotNil(t, preloads)
	assert.Contains(t, preloads, "Moves")
	assert.Contains(t, preloads, "Moves.Stops")
	assert.Contains(t, preloads, "Commodities")
	assert.Contains(t, preloads, "Commodities.Commodity")
}

func TestGetAvailableVariables_WithMultipleProperties(t *testing.T) {
	t.Parallel()

	builder, registry := setupEnvironmentBuilder(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-avail-vars",
		"type": "object",
		"x-formula-context": {
			"entityType": "Shipment"
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
			},
			"totalDistance": {
				"type": "number",
				"x-source": {
					"computed": true,
					"function": "computeTotalDistance"
				}
			}
		}
	}`

	err := registry.Register("test-avail-vars", []byte(schemaJSON))
	require.NoError(t, err)

	variables := builder.GetAvailableVariables("test-avail-vars")
	require.NotNil(t, variables)
	assert.Len(t, variables, 3)
}

func TestEnvironmentBuilder_Build_WithSchemaDirectFields(t *testing.T) {
	t.Parallel()

	builder, registry := setupEnvironmentBuilder(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-direct-fields",
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

	err := registry.Register("test-direct-fields", []byte(schemaJSON))
	require.NoError(t, err)

	entity := &TestShipment{
		Weight: 5000,
		Pieces: 100,
	}

	env, buildErr := builder.Build(entity, "test-direct-fields")
	require.NoError(t, buildErr)
	require.NotNil(t, env)

	assert.Equal(t, int64(5000), env["weight"])
	assert.Equal(t, int64(100), env["pieces"])
}

func TestEnvironmentBuilder_Build_WithComputedAndDirect(t *testing.T) {
	t.Parallel()

	builder, registry := setupEnvironmentBuilder(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-mixed-fields",
		"type": "object",
		"x-formula-context": {
			"entityType": "TestShipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": ["Moves"]
		},
		"properties": {
			"weight": {
				"type": "number",
				"x-source": {
					"field": "Weight"
				}
			},
			"totalDistance": {
				"type": "number",
				"x-source": {
					"computed": true,
					"function": "computeTotalDistance"
				}
			}
		}
	}`

	err := registry.Register("test-mixed-fields", []byte(schemaJSON))
	require.NoError(t, err)

	entity := &TestShipment{
		Weight: 3000,
		Moves: []Move{
			{Distance: 100.0},
			{Distance: 200.0},
		},
	}

	env, buildErr := builder.Build(entity, "test-mixed-fields")
	require.NoError(t, buildErr)
	require.NotNil(t, env)

	assert.Equal(t, int64(3000), env["weight"])
	assert.InDelta(t, 300.0, env["totalDistance"], 0.1)
}

func TestEnvironmentBuilder_Build_NullableFieldWithError(t *testing.T) {
	t.Parallel()

	builder, registry := setupEnvironmentBuilder(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-nullable-error",
		"type": "object",
		"x-formula-context": {
			"entityType": "TestShipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": []
		},
		"properties": {
			"missingField": {
				"type": "number",
				"x-source": {
					"field": "NonExistentField",
					"nullable": true
				}
			}
		}
	}`

	err := registry.Register("test-nullable-error", []byte(schemaJSON))
	require.NoError(t, err)

	entity := &TestShipment{Weight: 5000}

	env, buildErr := builder.Build(entity, "test-nullable-error")
	require.NoError(t, buildErr)
	require.NotNil(t, env)

	assert.Nil(t, env["missingField"])
}

func TestEnvironmentBuilder_Build_NonNullableFieldSkipped(t *testing.T) {
	t.Parallel()

	builder, registry := setupEnvironmentBuilder(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-non-nullable-skip",
		"type": "object",
		"x-formula-context": {
			"entityType": "TestShipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": []
		},
		"properties": {
			"missingField": {
				"type": "number",
				"x-source": {
					"field": "NonExistentField"
				}
			},
			"weight": {
				"type": "number",
				"x-source": {
					"field": "Weight"
				}
			}
		}
	}`

	err := registry.Register("test-non-nullable-skip", []byte(schemaJSON))
	require.NoError(t, err)

	entity := &TestShipment{Weight: 5000}

	env, buildErr := builder.Build(entity, "test-non-nullable-skip")
	require.NoError(t, buildErr)
	require.NotNil(t, env)

	_, exists := env["missingField"]
	assert.False(t, exists)
	assert.Equal(t, int64(5000), env["weight"])
}

func TestNewEnvironmentBuilder(t *testing.T) {
	t.Parallel()

	registry := schema.NewRegistry()

	builder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
	})

	require.NotNil(t, builder)
}
