package schema_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/pkg/formula/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaRegistry(t *testing.T) {
	registry := schema.NewSchemaRegistry()
	assert.NotNil(t, registry)
}

func TestSchemaRegistry_RegisterAndGet(t *testing.T) {
	registry := schema.NewSchemaRegistry()

	// * Test registering a new schema
	schemaJSON := `{
		"$id": "https://example.com/test-entity.schema.json",
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title": "Test Entity",
		"description": "Test entity for unit tests",
		"type": "object",
		"properties": {
			"id": {
				"type": "string",
				"description": "Entity ID"
			},
			"name": {
				"type": "string",
				"description": "Entity name",
				"x-source": {
					"path": "Name"
				}
			},
			"value": {
				"type": "number",
				"description": "Numeric value",
				"x-source": {
					"path": "Value",
					"transform": "float64"
				}
			}
		},
		"required": ["id", "name"],
		"x-formula-context": {
			"entity": "TestEntity",
			"version": "1.0.0"
		}
	}`

	err := registry.RegisterSchema("test-entity", []byte(schemaJSON))
	require.NoError(t, err)

	// * Test retrieving the registered schema
	def, err := registry.GetSchema("test-entity")
	require.NoError(t, err)
	assert.NotNil(t, def)
	assert.Equal(t, "Test Entity", def.Title)
	assert.Equal(t, "Test entity for unit tests", def.Description)
	
	// * Check field sources were extracted
	assert.Len(t, def.FieldSources, 2)
	assert.Contains(t, def.FieldSources, "name")
	assert.Contains(t, def.FieldSources, "value")
	
	// * Verify field source details
	nameSource := def.FieldSources["name"]
	assert.Equal(t, "Name", nameSource.Path)
	assert.Empty(t, nameSource.Transform)
	
	valueSource := def.FieldSources["value"]
	assert.Equal(t, "Value", valueSource.Path)
	assert.Equal(t, "float64", valueSource.Transform)
}

func TestSchemaRegistry_RegisterInvalidSchema(t *testing.T) {
	registry := schema.NewSchemaRegistry()
	
	tests := []struct {
		name        string
		schemaJSON  string
		wantErrMsg  string
	}{
		{
			name:       "invalid JSON",
			schemaJSON: `{invalid json`,
			wantErrMsg: "invalid schema JSON",
		},
		{
			name:       "missing required fields",
			schemaJSON: `{"type": "object"}`,
			wantErrMsg: "missing $id",
		},
		{
			name: "invalid property type",
			schemaJSON: `{
				"$id": "https://example.com/test.schema.json",
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"title": "Test",
				"type": "object",
				"properties": {
					"field": {
						"type": "invalid-type"
					}
				}
			}`,
			wantErrMsg: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.RegisterSchema(tt.name, []byte(tt.schemaJSON))
			assert.Error(t, err)
			if tt.wantErrMsg != "" {
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			}
		})
	}
}

func TestSchemaRegistry_GetNonExistent(t *testing.T) {
	registry := schema.NewSchemaRegistry()
	
	def, err := registry.GetSchema("non-existent")
	assert.Error(t, err)
	assert.Nil(t, def)
	assert.Contains(t, err.Error(), "schema not found")
}

func TestSchemaRegistry_ListSchemas(t *testing.T) {
	registry := schema.NewSchemaRegistry()
	
	// * Initially empty
	schemas := registry.ListSchemas()
	assert.Empty(t, schemas)
	
	// * Register multiple schemas
	schema1 := `{
		"$id": "https://example.com/entity1.schema.json",
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title": "Entity 1",
		"type": "object"
	}`
	
	schema2 := `{
		"$id": "https://example.com/entity2.schema.json",
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title": "Entity 2",
		"type": "object"
	}`
	
	err := registry.RegisterSchema("entity1", []byte(schema1))
	require.NoError(t, err)
	
	err = registry.RegisterSchema("entity2", []byte(schema2))
	require.NoError(t, err)
	
	// * List all schemas
	schemas = registry.ListSchemas()
	assert.Len(t, schemas, 2)
	
	// * Check that both schemas are present
	titles := make([]string, 0, len(schemas))
	for _, s := range schemas {
		titles = append(titles, s.Title)
	}
	assert.Contains(t, titles, "Entity 1")
	assert.Contains(t, titles, "Entity 2")
}

func TestSchemaRegistry_ComputedFields(t *testing.T) {
	registry := schema.NewSchemaRegistry()
	
	schemaJSON := `{
		"$id": "https://example.com/computed.schema.json",
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title": "Computed Fields Test",
		"type": "object",
		"properties": {
			"baseValue": {
				"type": "number",
				"x-source": {
					"path": "BaseValue"
				}
			},
			"computedValue": {
				"type": "number",
				"description": "Computed from base value",
				"x-source": {
					"computed": true,
					"function": "computeDouble"
				}
			},
			"derivedField": {
				"type": "string",
				"x-source": {
					"computed": true,
					"function": "deriveString",
					"dependsOn": ["baseValue"]
				}
			}
		},
		"x-data-source": {
			"entity": "ComputedEntity",
			"table": "computed_entities"
		}
	}`
	
	err := registry.RegisterSchema("computed", []byte(schemaJSON))
	require.NoError(t, err)
	
	def, err := registry.GetSchema("computed")
	require.NoError(t, err)
	
	// * Verify computed fields
	assert.Len(t, def.FieldSources, 3)
	
	// * Check regular field
	baseField := def.FieldSources["baseValue"]
	assert.False(t, baseField.Computed)
	assert.Equal(t, "BaseValue", baseField.Path)
	
	// * Check computed fields
	computedField := def.FieldSources["computedValue"]
	assert.True(t, computedField.Computed)
	assert.Equal(t, "computeDouble", computedField.Function)
	
	derivedField := def.FieldSources["derivedField"]
	assert.True(t, derivedField.Computed)
	assert.Equal(t, "deriveString", derivedField.Function)
}

func TestSchemaRegistry_NestedProperties(t *testing.T) {
	registry := schema.NewSchemaRegistry()
	
	schemaJSON := `{
		"$id": "https://example.com/nested.schema.json",
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title": "Nested Properties Test",
		"type": "object",
		"properties": {
			"id": {
				"type": "string"
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
					},
					"contact": {
						"type": "object",
						"properties": {
							"email": {
								"type": "string",
								"x-source": {
									"path": "Customer.Contact.Email"
								}
							}
						}
					}
				}
			},
			"items": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"quantity": {
							"type": "number",
							"x-source": {
								"path": "Items[].Quantity"
							}
						}
					}
				}
			}
		}
	}`
	
	err := registry.RegisterSchema("nested", []byte(schemaJSON))
	require.NoError(t, err)
	
	def, err := registry.GetSchema("nested")
	require.NoError(t, err)
	
	// * Verify nested field sources were extracted
	assert.Len(t, def.FieldSources, 4)
	assert.Contains(t, def.FieldSources, "customer.name")
	assert.Contains(t, def.FieldSources, "customer.code")
	assert.Contains(t, def.FieldSources, "customer.contact.email")
	assert.Contains(t, def.FieldSources, "items[].quantity")
	
	// * Check nested paths
	assert.Equal(t, "Customer.Name", def.FieldSources["customer.name"].Path)
	assert.Equal(t, "Customer.Code", def.FieldSources["customer.code"].Path)
	assert.Equal(t, "Customer.Contact.Email", def.FieldSources["customer.contact.email"].Path)
	assert.Equal(t, "Items[].Quantity", def.FieldSources["items[].quantity"].Path)
}

func TestSchemaRegistry_DataSourceExtension(t *testing.T) {
	registry := schema.NewSchemaRegistry()
	
	schemaJSON := `{
		"$id": "https://example.com/datasource.schema.json",
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title": "Data Source Test",
		"type": "object",
		"properties": {
			"id": {"type": "string"}
		},
		"x-data-source": {
			"entity": "TestEntity",
			"table": "test_entities",
			"joins": {
				"related": {
					"table": "related_entities",
					"on": "test_entities.related_id = related_entities.id",
					"type": "LEFT"
				}
			},
			"filters": [
				{
					"field": "status",
					"operator": "eq",
					"value": "active"
				},
				{
					"field": "created_at",
					"operator": "gte",
					"value": "now() - interval '30 days'"
				}
			],
			"preload": ["Customer", "Items"],
			"orderBy": "created_at DESC"
		}
	}`
	
	err := registry.RegisterSchema("datasource", []byte(schemaJSON))
	require.NoError(t, err)
	
	def, err := registry.GetSchema("datasource")
	require.NoError(t, err)
	
	// * Verify data source information
	assert.Equal(t, "TestEntity", def.DataSource.Entity)
	assert.Equal(t, "test_entities", def.DataSource.Table)
	assert.Equal(t, "created_at DESC", def.DataSource.OrderBy)
	
	// * Check joins
	assert.Len(t, def.DataSource.Joins, 1)
	relatedJoin := def.DataSource.Joins["related"]
	assert.Equal(t, "related_entities", relatedJoin.Table)
	assert.Equal(t, "test_entities.related_id = related_entities.id", relatedJoin.On)
	assert.Equal(t, "LEFT", relatedJoin.Type)
	
	// * Check filters
	assert.Len(t, def.DataSource.Filters, 2)
	assert.Equal(t, "status", def.DataSource.Filters[0].Field)
	assert.Equal(t, "eq", def.DataSource.Filters[0].Operator)
	assert.Equal(t, "active", def.DataSource.Filters[0].Value)
	
	// * Check preloads
	assert.Contains(t, def.DataSource.Preload, "Customer")
	assert.Contains(t, def.DataSource.Preload, "Items")
}

func TestSchemaRegistry_ConcurrentAccess(t *testing.T) {
	registry := schema.NewSchemaRegistry()
	
	// * Register initial schema
	schemaJSON := `{
		"$id": "https://example.com/concurrent.schema.json",
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title": "Concurrent Test",
		"type": "object"
	}`
	
	err := registry.RegisterSchema("concurrent", []byte(schemaJSON))
	require.NoError(t, err)
	
	// * Test concurrent reads and writes
	done := make(chan bool)
	errors := make(chan error, 10)
	
	// * Multiple readers
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				_, err := registry.GetSchema("concurrent")
				if err != nil {
					errors <- err
					return
				}
			}
		}()
	}
	
	// * Multiple writers (updating same schema)
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			newSchema := []byte(fmt.Sprintf(`{
				"$id": "https://example.com/concurrent%d.schema.json",
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"title": "Concurrent Test %d",
				"type": "object"
			}`, id, id))
			err := registry.RegisterSchema(fmt.Sprintf("concurrent%d", id), newSchema)
			if err != nil {
				errors <- err
			}
		}(i)
	}
	
	// * Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	close(errors)
	
	// * Check for errors
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

func TestSchemaRegistry_ValidationOfProperties(t *testing.T) {
	registry := schema.NewSchemaRegistry()
	
	// * Test schema with various validation rules
	schemaJSON := `{
		"$id": "https://example.com/validation.schema.json",
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"title": "Validation Test",
		"type": "object",
		"properties": {
			"age": {
				"type": "integer",
				"minimum": 0,
				"maximum": 150,
				"x-source": {
					"path": "Age"
				}
			},
			"email": {
				"type": "string",
				"format": "email",
				"x-source": {
					"path": "Email"
				}
			},
			"status": {
				"type": "string",
				"enum": ["active", "inactive", "pending"],
				"x-source": {
					"path": "Status"
				}
			},
			"tags": {
				"type": "array",
				"items": {
					"type": "string"
				},
				"minItems": 1,
				"maxItems": 10,
				"x-source": {
					"path": "Tags"
				}
			}
		},
		"required": ["email", "status"]
	}`
	
	err := registry.RegisterSchema("validation", []byte(schemaJSON))
	require.NoError(t, err)
	
	def, err := registry.GetSchema("validation")
	require.NoError(t, err)
	
	// * Verify all field sources were extracted
	assert.Len(t, def.FieldSources, 4)
	assert.Contains(t, def.FieldSources, "age")
	assert.Contains(t, def.FieldSources, "email")
	assert.Contains(t, def.FieldSources, "status")
	assert.Contains(t, def.FieldSources, "tags")
	
	// * The schema is valid and registered
	assert.Equal(t, "Validation Test", def.Title)
}