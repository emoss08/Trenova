package formula_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/pkg/formula/schema"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMissingSchemaIntegration(t *testing.T) {
	t.Run("schema is loaded but not used for variables", func(t *testing.T) {
		// Create and register a simple schema
		schemaJSON := []byte(`{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"$id": "test-schema",
			"properties": {
				"baseRate": {
					"description": "Base rate for calculation",
					"type": "number",
					"x-source": {
						"field": "base_rate",
						"path": "BaseRate",
						"transform": "decimalToFloat64"
					}
				},
				"distance": {
					"description": "Distance in miles",
					"type": "number",
					"x-source": {
						"field": "distance",
						"path": "Distance",
						"transform": "int64ToFloat64"
					}
				},
				"hasSpecialHandling": {
					"description": "Requires special handling",
					"type": "boolean",
					"x-source": {
						"computed": true,
						"function": "computeHasSpecialHandling"
					}
				}
			}
		}`)

		// Register the schema
		registry := schema.NewSchemaRegistry()
		err := registry.RegisterSchema("test", schemaJSON)
		require.NoError(t, err)

		// Get the registered schema
		testSchema, err := registry.GetSchema("test")
		require.NoError(t, err)

		// Verify schema has extracted field sources
		assert.NotNil(t, testSchema.FieldSources)
		assert.Len(t, testSchema.FieldSources, 3)

		// Check that field sources were extracted correctly
		baseRateSource := testSchema.FieldSources["baseRate"]
		require.NotNil(t, baseRateSource)
		assert.Equal(t, "BaseRate", baseRateSource.Path)
		assert.Equal(t, "decimalToFloat64", baseRateSource.Transform)

		// THE PROBLEM: These schema fields are NOT available as variables
		varRegistry := variables.NewRegistry()

		// Try to get a variable that should exist from schema
		_, err = varRegistry.Get("baseRate")
		assert.Error(t, err, "baseRate variable doesn't exist - schema not integrated")

		_, err = varRegistry.Get("distance")
		assert.Error(t, err, "distance variable doesn't exist - schema not integrated")
	})

	t.Run("field sources exist but aren't used by variable context", func(t *testing.T) {
		// Create test entity
		type TestEntity struct {
			BaseRate int64
			Distance int64
		}

		entity := &TestEntity{
			BaseRate: 250, // $2.50 as cents
			Distance: 100,
		}

		// Create resolver and context
		resolver := schema.NewDefaultDataResolver()
		varCtx := variables.NewDefaultContext(entity, resolver)

		// This works with simple path
		value, err := varCtx.GetField("BaseRate")
		require.NoError(t, err)
		assert.Equal(t, int64(250), value)

		// But it doesn't use schema transformations
		// Schema says this should be transformed to float64
		// but the resolver doesn't know about the schema
	})

	t.Run("what needs to be implemented", func(t *testing.T) {
		// 1. A function to register variables from schema
		// Something like:
		// func RegisterVariablesFromSchema(varRegistry *variables.Registry, schemaRegistry *schema.SchemaRegistry, schemaID string) error

		// 2. A schema-aware variable context that:
		// - Uses schema field sources for resolution
		// - Applies transformations automatically
		// - Knows about computed fields

		// 3. Integration in the formula service to:
		// - Load entity with proper preloads from schema
		// - Create context with schema awareness
		// - Make all schema fields available as variables
	})
}

func TestSchemaFieldSourceUsage(t *testing.T) {
	t.Run("demonstrate field source extraction", func(t *testing.T) {
		registry := schema.NewSchemaRegistry()

		// Register a nested schema
		schemaJSON := []byte(`{
			"$id": "nested-schema",
			"properties": {
				"customer": {
					"type": "object",
					"x-source": {
						"relation": "Customer"
					},
					"properties": {
						"name": {
							"type": "string",
							"x-source": {
								"path": "Customer.Name"
							}
						},
						"discountRate": {
							"type": "number",
							"x-source": {
								"path": "Customer.DiscountRate",
								"transform": "decimalToFloat64"
							}
						}
					}
				}
			}
		}`)

		err := registry.RegisterSchema("nested", schemaJSON)
		require.NoError(t, err)

		nestedSchema, err := registry.GetSchema("nested")
		require.NoError(t, err)

		// Field sources are extracted including nested ones
		assert.Contains(t, nestedSchema.FieldSources, "customer.name")
		assert.Contains(t, nestedSchema.FieldSources, "customer.discountRate")

		// But these aren't accessible as variables!
	})
}

// This test shows what the complete implementation would look like
func TestIdealSchemaIntegration(t *testing.T) {
	t.Run("proposed solution", func(t *testing.T) {
		// Step 1: Register schema
		schemaRegistry := schema.NewSchemaRegistry()
		schemaJSON := []byte(`{
			"$id": "pricing-schema",
			"properties": {
				"weight": {
					"type": "number",
					"description": "Weight in pounds",
					"x-source": {
						"path": "Weight",
						"transform": "int64ToFloat64"
					}
				},
				"rate": {
					"type": "number", 
					"description": "Rate per pound",
					"x-source": {
						"path": "RatePerPound",
						"transform": "decimalToFloat64"
					}
				}
			}
		}`)

		err := schemaRegistry.RegisterSchema("pricing", schemaJSON)
		require.NoError(t, err)

		// Step 2: What's missing - Register variables from schema
		// This would create Variable instances for each schema field
		// varRegistry := variables.NewRegistry()
		// RegisterSchemaVariables(varRegistry, schemaRegistry, "pricing")

		// Step 3: Create schema-aware context
		// This would use schema field sources for resolution
		// ctx := NewSchemaAwareContext(entity, schemaRegistry, "pricing")

		// Step 4: Evaluate formula with all schema fields available
		// evaluator.Evaluate(ctx, "weight * rate", ctx)
	})
}
