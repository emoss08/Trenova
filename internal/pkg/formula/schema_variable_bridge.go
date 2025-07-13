package formula

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/types/formula"
	"github.com/emoss08/trenova/internal/pkg/formula/schema"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
)

// SchemaVariableBridge connects schemas to the variable system
type SchemaVariableBridge struct {
	schemaRegistry *schema.SchemaRegistry
	varRegistry    *variables.Registry
}

// NewSchemaVariableBridge creates a new bridge
func NewSchemaVariableBridge(
	schemaRegistry *schema.SchemaRegistry,
	varRegistry *variables.Registry,
) *SchemaVariableBridge {
	return &SchemaVariableBridge{
		schemaRegistry: schemaRegistry,
		varRegistry:    varRegistry,
	}
}

// RegisterSchemaVariables creates variables from a schema definition
func (b *SchemaVariableBridge) RegisterSchemaVariables(schemaID string) error {
	// Get the schema
	schemaDef, err := b.schemaRegistry.GetSchema(schemaID)
	if err != nil {
		return fmt.Errorf("failed to get schema %s: %w", schemaID, err)
	}

	// Register a variable for each field source
	for fieldName, fieldSource := range schemaDef.FieldSources {
		// Create variable using the proper constructor
		source := variables.VariableSource(schemaID)

		// Capture fieldSource in closure
		fs := fieldSource
		resolver := func(ctx variables.VariableContext) (any, error) {
			// Use schema-aware resolution
			return resolveFromSchema(ctx, fs)
		}

		varDef := variables.NewVariable(
			fieldName,
			getFieldDescription(schemaDef, fieldName),
			inferTypeFromSource(fieldSource),
			source,
			resolver,
		)

		// Register the variable
		if err := b.varRegistry.Register(varDef); err != nil {
			return fmt.Errorf("failed to register variable %s: %w", fieldName, err)
		}

		// Also register flattened field names for nested objects
		if strings.Contains(fieldName, ".") {
			flatName := strings.ToLower(strings.ReplaceAll(fieldName, ".", ""))
			varDefFlat := variables.NewVariable(
				flatName,
				getFieldDescription(schemaDef, fieldName),
				inferTypeFromSource(fieldSource),
				source,
				resolver,
			)
			if err := b.varRegistry.Register(varDefFlat); err != nil {
				// Don't fail if flattened name conflicts, just skip
				continue
			}
		}
	}

	return nil
}

// GetRequiredData analyzes an expression to determine what data needs to be loaded
func (b *SchemaVariableBridge) GetRequiredData(expression string) (*DataRequirements, error) {
	// This would parse the expression and determine:
	// 1. What variables are used
	// 2. What schema fields those map to
	// 3. What preloads are required

	req := &DataRequirements{
		Fields:   make(map[string]bool),
		Preloads: make(map[string]bool),
	}

	// TODO: Implement expression parsing to extract variables
	// For now, return empty requirements
	return req, nil
}

// DataRequirements describes what data needs to be loaded
type DataRequirements struct {
	Fields   map[string]bool // Direct fields needed
	Preloads map[string]bool // Relations to preload
}

// SchemaAwareContext is a variable context that uses schema for resolution
type SchemaAwareContext struct {
	entity         any
	schemaID       string
	schemaRegistry *schema.SchemaRegistry
	resolver       *schema.DefaultDataResolver
	cache          map[string]any
}

// NewSchemaAwareContext creates a context that uses schema definitions
func NewSchemaAwareContext(
	entity any,
	schemaID string,
	schemaRegistry *schema.SchemaRegistry,
	resolver *schema.DefaultDataResolver,
) *SchemaAwareContext {
	return &SchemaAwareContext{
		entity:         entity,
		schemaID:       schemaID,
		schemaRegistry: schemaRegistry,
		resolver:       resolver,
		cache:          make(map[string]any),
	}
}

// GetEntity returns the primary entity
func (c *SchemaAwareContext) GetEntity() any {
	return c.entity
}

// GetField retrieves a field value using schema definitions
func (c *SchemaAwareContext) GetField(path string) (any, error) {
	// Check cache first
	if val, ok := c.cache[path]; ok {
		return val, nil
	}

	// Get schema
	schemaDef, err := c.schemaRegistry.GetSchema(c.schemaID)
	if err != nil {
		return nil, err
	}

	// Find field source in schema
	fieldSource, ok := schemaDef.FieldSources[path]
	if !ok {
		// Try with different casing or nested paths
		fieldSource = findFieldSource(schemaDef, path)
		if fieldSource == nil {
			return nil, fmt.Errorf("field %s not found in schema", path)
		}
	}

	// Resolve using schema-defined source
	val, err := c.resolver.ResolveField(c.entity, fieldSource)
	if err != nil {
		return nil, err
	}

	// Cache the result
	c.cache[path] = val
	return val, nil
}

// GetComputed retrieves a computed value
func (c *SchemaAwareContext) GetComputed(function string) (any, error) {
	return c.resolver.ResolveComputed(c.entity, &schema.FieldSource{
		Computed: true,
		Function: function,
	})
}

// GetMetadata returns context metadata
func (c *SchemaAwareContext) GetMetadata() map[string]any {
	return map[string]any{
		"schema": c.schemaID,
		"entity": c.entity,
	}
}

// GetFieldSources returns all available fields from schema
func (c *SchemaAwareContext) GetFieldSources() map[string]any {
	schemaDef, err := c.schemaRegistry.GetSchema(c.schemaID)
	if err != nil {
		return nil
	}

	result := make(map[string]any)
	for name := range schemaDef.FieldSources {
		if val, err := c.GetField(name); err == nil {
			result[name] = val
		}
	}
	return result
}

// Helper functions

func inferTypeFromSource(source *schema.FieldSource) formula.ValueType {
	// Infer type from transform function or default to Any
	if source.Transform != "" {
		switch source.Transform {
		case "decimalToFloat64", "int64ToFloat64", "int16ToFloat64":
			return formula.ValueTypeNumber
		}
	}

	if source.Computed {
		// Computed fields need specific type mapping
		switch source.Function {
		case "computeTemperatureDifferential", "computeTotalCommodityWeight":
			return formula.ValueTypeNumber
		case "computeHasHazmat", "computeRequiresTemperatureControl":
			return formula.ValueTypeBoolean
		case "computeTotalStops":
			return formula.ValueTypeNumber
		}
	}

	// Default to Any for flexibility
	return formula.ValueTypeAny
}

func getFieldDescription(schemaDef *schema.SchemaDefinition, fieldName string) string {
	// Extract description from property schema
	parts := strings.Split(fieldName, ".")
	props := schemaDef.Properties

	for i, part := range parts {
		if prop, ok := props[part]; ok {
			if i == len(parts)-1 {
				return prop.Description
			}
			// Navigate nested properties
			if prop.Properties != nil {
				props = prop.Properties
			}
		}
	}

	return ""
}

func resolveFromSchema(ctx variables.VariableContext, source *schema.FieldSource) (any, error) {
	// This connects variable resolution to schema field sources
	if source.Computed {
		return ctx.GetComputed(source.Function)
	}

	// For schema-aware context, use the field source directly to avoid lookup issues
	if schemaCtx, ok := ctx.(*SchemaAwareContext); ok {
		return schemaCtx.resolver.ResolveField(schemaCtx.entity, source)
	}

	return ctx.GetField(source.Path)
}

func findFieldSource(schemaDef *schema.SchemaDefinition, path string) *schema.FieldSource {
	// Try exact match by field name (schema key)
	if source, ok := schemaDef.FieldSources[path]; ok {
		return source
	}

	// Try case-insensitive match by field name
	lowerPath := strings.ToLower(path)
	for name, source := range schemaDef.FieldSources {
		if strings.ToLower(name) == lowerPath {
			return source
		}
	}

	// Try matching by field source path (database field name -> schema field source)
	for _, source := range schemaDef.FieldSources {
		if source.Path == path {
			return source
		}
	}

	// Try matching by database field name (e.g., path="temperature_max" matches source.Field="temperature_max")
	for _, source := range schemaDef.FieldSources {
		if source.Field == path {
			return source
		}
	}

	// Try case-insensitive match by field source path
	for _, source := range schemaDef.FieldSources {
		if strings.ToLower(source.Path) == lowerPath {
			return source
		}
	}

	// Try case-insensitive match by database field name
	for _, source := range schemaDef.FieldSources {
		if strings.ToLower(source.Field) == lowerPath {
			return source
		}
	}

	return nil
}
