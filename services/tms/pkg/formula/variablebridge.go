package formula

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/formula/schema"
	"github.com/emoss08/trenova/pkg/formula/variables"
	"github.com/emoss08/trenova/pkg/formulatypes"
)

type SchemaVariableBridge struct {
	schemaRegistry *schema.Registry
	varRegistry    *variables.Registry
}

func NewSchemaVariableBridge(
	schemaRegistry *schema.Registry,
	varRegistry *variables.Registry,
) *SchemaVariableBridge {
	return &SchemaVariableBridge{
		schemaRegistry: schemaRegistry,
		varRegistry:    varRegistry,
	}
}

func (b *SchemaVariableBridge) RegisterSchemaVariables(schemaID string) error {
	// Get the schema
	schemaDef, err := b.schemaRegistry.GetSchema(schemaID)
	if err != nil {
		return fmt.Errorf("failed to get schema %s: %w", schemaID, err)
	}

	for fieldName, fieldSource := range schemaDef.FieldSources {
		source := variables.VariableSource(schemaID)

		fs := fieldSource
		resolver := func(ctx variables.VariableContext) (any, error) {
			return resolveFromSchema(ctx, fs)
		}

		varDef := variables.NewVariable(
			fieldName,
			getFieldDescription(schemaDef, fieldName),
			inferTypeFromSource(fieldSource),
			source,
			resolver,
		)

		if err := b.varRegistry.Register(varDef); err != nil {
			return fmt.Errorf("failed to register variable %s: %w", fieldName, err)
		}

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
				continue
			}
		}
	}

	return nil
}

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

type DataRequirements struct {
	Fields   map[string]bool
	Preloads map[string]bool
}

type SchemaAwareContext struct {
	entity         any
	schemaID       string
	schemaRegistry *schema.Registry
	resolver       *schema.DefaultDataResolver
	cache          map[string]any
}

func NewSchemaAwareContext(
	entity any,
	schemaID string,
	schemaRegistry *schema.Registry,
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

func (c *SchemaAwareContext) GetEntity() any {
	return c.entity
}

func (c *SchemaAwareContext) GetField(path string) (any, error) {
	if val, ok := c.cache[path]; ok {
		return val, nil
	}

	schemaDef, err := c.schemaRegistry.GetSchema(c.schemaID)
	if err != nil {
		return nil, err
	}

	fieldSource, ok := schemaDef.FieldSources[path]
	if !ok {
		fieldSource = findFieldSource(schemaDef, path)
		if fieldSource == nil {
			return nil, fmt.Errorf("field %s not found in schema", path)
		}
	}

	val, err := c.resolver.ResolveField(c.entity, fieldSource)
	if err != nil {
		return nil, err
	}

	c.cache[path] = val
	return val, nil
}

func (c *SchemaAwareContext) GetComputed(function string) (any, error) {
	return c.resolver.ResolveComputed(c.entity, &schema.FieldSource{
		Computed: true,
		Function: function,
	})
}

func (c *SchemaAwareContext) GetMetadata() map[string]any {
	return map[string]any{
		"schema": c.schemaID,
		"entity": c.entity,
	}
}

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

func inferTypeFromSource(source *schema.FieldSource) formulatypes.ValueType {
	if source.Transform != "" {
		switch source.Transform {
		case "decimalToFloat64", "int64ToFloat64", "int16ToFloat64":
			return formulatypes.ValueTypeNumber
		}
	}

	if source.Computed {
		switch source.Function {
		case "computeTemperatureDifferential", "computeTotalCommodityWeight":
			return formulatypes.ValueTypeNumber
		case "computeHasHazmat", "computeRequiresTemperatureControl":
			return formulatypes.ValueTypeBoolean
		case "computeTotalStops":
			return formulatypes.ValueTypeNumber
		}
	}

	return formulatypes.ValueTypeAny
}

func getFieldDescription(schemaDef *schema.Definition, fieldName string) string {
	parts := strings.Split(fieldName, ".")
	props := schemaDef.Properties

	for i, part := range parts {
		if prop, ok := props[part]; ok {
			if i == len(parts)-1 {
				return prop.Description
			}
			if prop.Properties != nil {
				props = prop.Properties
			}
		}
	}

	return ""
}

func resolveFromSchema(ctx variables.VariableContext, source *schema.FieldSource) (any, error) {
	if source.Computed {
		return ctx.GetComputed(source.Function)
	}

	if schemaCtx, ok := ctx.(*SchemaAwareContext); ok {
		return schemaCtx.resolver.ResolveField(schemaCtx.entity, source)
	}

	return ctx.GetField(source.Path)
}

func findFieldSource(schemaDef *schema.Definition, path string) *schema.FieldSource {
	if source, ok := schemaDef.FieldSources[path]; ok {
		return source
	}

	lowerPath := strings.ToLower(path)
	for name, source := range schemaDef.FieldSources {
		if strings.ToLower(name) == lowerPath {
			return source
		}
	}

	for _, source := range schemaDef.FieldSources {
		if source.Path == path {
			return source
		}
	}

	for _, source := range schemaDef.FieldSources {
		if source.Field == path {
			return source
		}
	}

	for _, source := range schemaDef.FieldSources {
		if strings.ToLower(source.Path) == lowerPath {
			return source
		}
	}

	for _, source := range schemaDef.FieldSources {
		if strings.ToLower(source.Field) == lowerPath {
			return source
		}
	}

	return nil
}
