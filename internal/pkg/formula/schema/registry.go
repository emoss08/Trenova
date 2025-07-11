package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/pkg/formula/errors"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// SchemaRegistry manages JSON schemas for formula contexts
//
//nolint:revive // this is fine
type SchemaRegistry struct {
	mu       sync.RWMutex
	compiler *jsonschema.Compiler
	schemas  map[string]*SchemaDefinition
}

// SchemaDefinition represents a schema with metadata
//
//nolint:revive // this is fine
type SchemaDefinition struct {
	ID          string                    `json:"$id"`
	Schema      string                    `json:"$schema"`
	Title       string                    `json:"title"`
	Description string                    `json:"description"`
	Type        string                    `json:"type"`
	Properties  map[string]PropertySchema `json:"properties"`
	Required    []string                  `json:"required,omitempty"`
	Version     string                    `json:"version"`

	// * Formula-specific extensions
	FormulaContext FormulaContextExtension `json:"x-formula-context"` //nolint:tagliatelle // this is fine
	DataSource     DataSource              `json:"x-data-source"`     //nolint:tagliatelle // this is fine

	// * Extracted field sources for easy access
	FieldSources map[string]*FieldSource

	// * Compiled schema for validation
	compiled *jsonschema.Schema
}

// PropertySchema represents a property in the schema
type PropertySchema struct {
	Type        any                       `json:"type"` // Can be string or array of strings
	Description string                    `json:"description"`
	Enum        []string                  `json:"enum,omitempty"`
	Minimum     *float64                  `json:"minimum,omitempty"`
	Maximum     *float64                  `json:"maximum,omitempty"`
	MinItems    *int                      `json:"minItems,omitempty"`
	MaxItems    *int                      `json:"maxItems,omitempty"`
	Properties  map[string]PropertySchema `json:"properties,omitempty"` // For object types
	Items       *PropertySchema           `json:"items,omitempty"`      // For array types
	Source      FieldSource               `json:"x-source"`             // Source information
}

// NewSchemaRegistry creates a new schema registry
func NewSchemaRegistry() *SchemaRegistry {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)

	return &SchemaRegistry{
		compiler: compiler,
		schemas:  make(map[string]*SchemaDefinition),
	}
}

// RegisterSchema registers a new schema
func (r *SchemaRegistry) RegisterSchema(id string, schemaJSON []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var schemaDef SchemaDefinition
	if err := sonic.Unmarshal(schemaJSON, &schemaDef); err != nil {
		return errors.NewSchemaError(id, "unmarshal", err)
	}

	if schemaDef.ID == "" {
		return errors.NewSchemaError(id, "validation", fmt.Errorf("missing $id field in schema"))
	}

	// * Use jsonschema.UnmarshalJSON to properly parse the schema for the compiler
	schema, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return errors.NewSchemaError(schemaDef.ID, "parse", err)
	}

	// * Add schema to compiler
	if err = r.compiler.AddResource(schemaDef.ID, schema); err != nil {
		return errors.NewSchemaError(schemaDef.ID, "add to compiler", err)
	}

	// * Compile the schema
	compiled, err := r.compiler.Compile(schemaDef.ID)
	if err != nil {
		return errors.NewSchemaError(schemaDef.ID, "compile", err)
	}
	schemaDef.compiled = compiled

	// * Extract field sources from properties
	schemaDef.FieldSources = r.extractFieldSources(schemaDef.Properties, "")

	// * Store by the simple ID provided by caller
	r.schemas[id] = &schemaDef
	return nil
}

// GetSchema retrieves a schema by ID
func (r *SchemaRegistry) GetSchema(id string) (*SchemaDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, exists := r.schemas[id]
	if !exists {
		return nil, fmt.Errorf("schema not found: %s", id)
	}
	return schema, nil
}

// ValidateData validates data against a schema
func (r *SchemaRegistry) ValidateData(schemaID string, data any) error {
	schema, err := r.GetSchema(schemaID)
	if err != nil {
		return err
	}

	if err = schema.compiled.Validate(data); err != nil {
		return errors.NewSchemaError(schemaID, "validate", err)
	}
	return nil
}

// ValidateJSON validates JSON data against a schema
func (r *SchemaRegistry) ValidateJSON(schemaID string, jsonData []byte) error {
	var data any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return r.ValidateData(schemaID, data)
}

// ListSchemas returns all registered schema definitions
func (r *SchemaRegistry) ListSchemas() []*SchemaDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make([]*SchemaDefinition, 0, len(r.schemas))
	for _, schema := range r.schemas {
		schemas = append(schemas, schema)
	}
	return schemas
}

// GetSchemasForEntity returns schemas applicable to a specific entity
func (r *SchemaRegistry) GetSchemasForEntity(entity string) []*SchemaDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var schemas []*SchemaDefinition
	for _, schema := range r.schemas {
		if slices.Contains(schema.FormulaContext.Entities, entity) {
			schemas = append(schemas, schema)
		}
	}
	return schemas
}

// extractFieldSources recursively extracts field sources from properties
func (r *SchemaRegistry) extractFieldSources( //nolint:gocognit // this is fine
	properties map[string]PropertySchema,
	prefix string,
) map[string]*FieldSource {
	sources := make(map[string]*FieldSource)

	// TODO(Wolfred): this is a heavy operation and each iteration copies 280 bytes we should consider a pointer or indexing
	for name, prop := range properties {
		fieldPath := name
		if prefix != "" {
			fieldPath = prefix + "." + name
		}

		// * Extract source if present
		if prop.Source.Path != "" || prop.Source.Computed {
			source := prop.Source // Copy the embedded source
			sources[fieldPath] = &source
		}

		// * Recursively process nested objects
		if prop.Properties != nil {
			nestedSources := r.extractFieldSources(prop.Properties, fieldPath)
			maps.Copy(sources, nestedSources)
		}

		// * Handle array items
		if prop.Type == "array" && prop.Items != nil {
			// * For array items, add [] to indicate array access
			arrayPrefix := fieldPath + "[]"
			if prop.Items.Source.Path != "" || prop.Items.Source.Computed {
				source := prop.Items.Source
				sources[arrayPrefix] = &source
			}
			// * Process properties of array items
			if prop.Items.Properties != nil {
				nestedSources := r.extractFieldSources(prop.Items.Properties, arrayPrefix)
				maps.Copy(sources, nestedSources)
			}
		}
	}

	return sources
}
