// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/pkg/formula/expression"
	"github.com/emoss08/trenova/internal/pkg/formula/ports"
	"github.com/emoss08/trenova/internal/pkg/formula/schema"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
)

// FormulaEvaluationService handles formula template evaluation with database integration
type FormulaEvaluationService struct {
	dataLoader     ports.DataLoader
	schemaRegistry *schema.SchemaRegistry
	varRegistry    *variables.Registry
	evaluator      *expression.Evaluator
	resolver       *schema.DefaultDataResolver
}

// NewFormulaEvaluationService creates a new service
func NewFormulaEvaluationService(
	dataLoader ports.DataLoader,
	schemaRegistry *schema.SchemaRegistry,
	varRegistry *variables.Registry,
	resolver *schema.DefaultDataResolver,
) *FormulaEvaluationService {
	return &FormulaEvaluationService{
		dataLoader:     dataLoader,
		schemaRegistry: schemaRegistry,
		varRegistry:    varRegistry,
		evaluator:      expression.NewEvaluator(varRegistry),
		resolver:       resolver,
	}
}

// EvaluateFormula evaluates a formula expression for a specific entity
func (s *FormulaEvaluationService) EvaluateFormula(
	ctx context.Context,
	formula string,
	entityType string,
	entityID string,
) (float64, error) {
	// Step 1: Analyze formula to determine data requirements
	requirements, err := s.analyzeFormulaRequirements(formula)
	if err != nil {
		return 0, fmt.Errorf("failed to analyze formula: %w", err)
	}

	// Step 2: Load entity data based on requirements
	entity, err := s.dataLoader.LoadEntityWithRequirements(ctx, entityType, entityID, requirements)
	if err != nil {
		return 0, fmt.Errorf("failed to load entity: %w", err)
	}

	// Step 3: Create schema-aware context
	varContext := NewSchemaAwareVariableContext(entity, entityType, s.schemaRegistry, s.resolver)

	// Step 4: Evaluate formula
	result, err := s.evaluator.Evaluate(ctx, formula, varContext)
	if err != nil {
		return 0, fmt.Errorf("evaluation failed: %w", err)
	}

	return result, nil
}

// analyzeFormulaRequirements determines what data is needed for the formula
func (s *FormulaEvaluationService) analyzeFormulaRequirements(
	formula string,
) (*ports.DataRequirements, error) {
	requirements := &ports.DataRequirements{
		Fields:         []string{},
		Preloads:       []string{},
		ComputedFields: []string{},
	}

	// Parse the formula to extract variable names
	variableNames := s.extractVariableNames(formula)

	// For each variable, determine what data is needed
	for _, varName := range variableNames {
		// Check if it's a nested path (e.g., "customer.name")
		if strings.Contains(varName, ".") {
			parts := strings.Split(varName, ".")
			// First part is likely a relation that needs preloading
			requirements.Preloads = append(requirements.Preloads, parts[0])
		} else {
			// Check if it's a computed field
			variable, err := s.varRegistry.Get(varName)
			if err == nil && s.isComputedVariable(variable) {
				requirements.ComputedFields = append(requirements.ComputedFields, varName)
			} else {
				requirements.Fields = append(requirements.Fields, varName)
			}
		}
	}

	// Remove duplicates
	requirements.Preloads = uniqueStrings(requirements.Preloads)
	requirements.Fields = uniqueStrings(requirements.Fields)
	requirements.ComputedFields = uniqueStrings(requirements.ComputedFields)

	return requirements, nil
}

// extractVariableNames extracts variable names from a formula expression
func (s *FormulaEvaluationService) extractVariableNames(formula string) []string {
	// Simple implementation - in production, use the expression parser
	var variables []string

	// Tokenize the formula
	tokenizer := expression.NewTokenizer(formula)
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		return variables
	}

	// Look for identifier tokens
	for _, token := range tokens {
		if token.Type == expression.TokenIdentifier {
			variables = append(variables, token.Value)
		}
	}

	return variables
}

// isComputedVariable checks if a variable is computed
func (s *FormulaEvaluationService) isComputedVariable(v variables.Variable) bool {
	// Check if the variable name matches known computed functions
	computedFunctions := []string{
		"temperatureDifferential",
		"hasHazmat",
		"requiresTemperatureControl",
		"totalStops",
	}

	for _, fn := range computedFunctions {
		if v.Name() == fn {
			return true
		}
	}

	return false
}

// SchemaAwareVariableContext implements VariableContext using schema definitions
type SchemaAwareVariableContext struct {
	entity         any
	entityType     string
	schemaRegistry *schema.SchemaRegistry
	resolver       *schema.DefaultDataResolver
}

// NewSchemaAwareVariableContext creates a new schema-aware context
func NewSchemaAwareVariableContext(
	entity any,
	entityType string,
	schemaRegistry *schema.SchemaRegistry,
	resolver *schema.DefaultDataResolver,
) *SchemaAwareVariableContext {
	return &SchemaAwareVariableContext{
		entity:         entity,
		entityType:     entityType,
		schemaRegistry: schemaRegistry,
		resolver:       resolver,
	}
}

// GetEntity returns the primary entity
func (c *SchemaAwareVariableContext) GetEntity() any {
	return c.entity
}

// GetField retrieves a field value by path
func (c *SchemaAwareVariableContext) GetField(path string) (any, error) {
	// Get schema to find field source
	schemaDef, err := c.schemaRegistry.GetSchema(c.entityType)
	if err != nil {
		return nil, fmt.Errorf("schema not found: %s", c.entityType)
	}

	// Find field source - first try exact match by field name
	fieldSource, ok := schemaDef.FieldSources[path]
	if !ok {
		// Try case-insensitive match by field name
		for name, source := range schemaDef.FieldSources {
			if strings.EqualFold(name, path) {
				fieldSource = source
				ok = true
				break
			}
		}

		// Try matching by field source path (e.g., path="temperature_max" matches source.Path="temperature_max")
		if !ok {
			for _, source := range schemaDef.FieldSources {
				if source.Path == path {
					fieldSource = source
					ok = true
					break
				}
			}
		}

		// Try case-insensitive match by field source path
		if !ok {
			for _, source := range schemaDef.FieldSources {
				if strings.EqualFold(source.Path, path) {
					fieldSource = source
					ok = true
					break
				}
			}
		}

		if !ok {
			return nil, fmt.Errorf("field not found in schema: %s", path)
		}
	}

	// Resolve using schema field source
	return c.resolver.ResolveField(c.entity, fieldSource)
}

// GetComputed retrieves a computed value
func (c *SchemaAwareVariableContext) GetComputed(function string) (any, error) {
	return c.resolver.ResolveComputed(c.entity, &schema.FieldSource{
		Computed: true,
		Function: function,
	})
}

// GetMetadata returns context metadata
func (c *SchemaAwareVariableContext) GetMetadata() map[string]any {
	return map[string]any{
		"entityType": c.entityType,
		"entity":     c.entity,
	}
}

// GetFieldSources returns available fields from the entity
func (c *SchemaAwareVariableContext) GetFieldSources() map[string]any {
	result := make(map[string]any)

	// Get schema
	schemaDef, err := c.schemaRegistry.GetSchema(c.entityType)
	if err != nil {
		return result
	}

	// Try to get each field
	for name := range schemaDef.FieldSources {
		if value, err := c.GetField(name); err == nil {
			result[name] = value
		}
	}

	return result
}

// Helper function to remove duplicate strings
func uniqueStrings(input []string) []string {
	keys := make(map[string]bool)
	list := []string{}

	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}

	return list
}
