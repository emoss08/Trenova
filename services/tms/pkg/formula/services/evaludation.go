package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/formula/expression"
	"github.com/emoss08/trenova/pkg/formula/ports"
	"github.com/emoss08/trenova/pkg/formula/schema"
	"github.com/emoss08/trenova/pkg/formula/variables"
)

type FormulaEvaluationService struct {
	dataLoader     ports.DataLoader
	schemaRegistry *schema.Registry
	varRegistry    *variables.Registry
	evaluator      *expression.Evaluator
	resolver       *schema.DefaultDataResolver
}

func NewFormulaEvaluationService(
	dataLoader ports.DataLoader,
	schemaRegistry *schema.Registry,
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

func (s *FormulaEvaluationService) EvaluateFormula(
	ctx context.Context,
	formula string,
	entityType string,
	entityID string,
) (float64, error) {
	requirements := s.analyzeFormulaRequirements(formula)

	entity, err := s.dataLoader.LoadEntityWithRequirements(ctx, entityType, entityID, requirements)
	if err != nil {
		return 0, fmt.Errorf("failed to load entity: %w", err)
	}

	varContext := NewSchemaAwareVariableContext(entity, entityType, s.schemaRegistry, s.resolver)

	result, err := s.evaluator.Evaluate(ctx, formula, varContext)
	if err != nil {
		return 0, fmt.Errorf("evaluation failed: %w", err)
	}

	return result, nil
}

func (s *FormulaEvaluationService) analyzeFormulaRequirements(
	formula string,
) *ports.DataRequirements {
	requirements := &ports.DataRequirements{
		Fields:         []string{},
		Preloads:       []string{},
		ComputedFields: []string{},
	}

	variableNames := s.extractVariableNames(formula)

	for _, varName := range variableNames {
		if strings.Contains(varName, ".") {
			parts := strings.Split(varName, ".")
			requirements.Preloads = append(requirements.Preloads, parts[0])
		} else {
			variable, err := s.varRegistry.Get(varName)
			if err == nil && s.isComputedVariable(variable) {
				requirements.ComputedFields = append(requirements.ComputedFields, varName)
			} else {
				requirements.Fields = append(requirements.Fields, varName)
			}
		}
	}

	requirements.Preloads = uniqueStrings(requirements.Preloads)
	requirements.Fields = uniqueStrings(requirements.Fields)
	requirements.ComputedFields = uniqueStrings(requirements.ComputedFields)

	return requirements
}

func (s *FormulaEvaluationService) extractVariableNames(formula string) []string {
	var varNames []string

	tokenizer := expression.NewTokenizer(formula)
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		return varNames
	}

	for _, token := range tokens {
		if token.Type == expression.TokenIdentifier {
			varNames = append(varNames, token.Value)
		}
	}

	return varNames
}

func (s *FormulaEvaluationService) isComputedVariable(v variables.Variable) bool {
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

type SchemaAwareVariableContext struct {
	entity         any
	entityType     string
	schemaRegistry *schema.Registry
	resolver       *schema.DefaultDataResolver
}

func NewSchemaAwareVariableContext(
	entity any,
	entityType string,
	schemaRegistry *schema.Registry,
	resolver *schema.DefaultDataResolver,
) *SchemaAwareVariableContext {
	return &SchemaAwareVariableContext{
		entity:         entity,
		entityType:     entityType,
		schemaRegistry: schemaRegistry,
		resolver:       resolver,
	}
}

func (c *SchemaAwareVariableContext) GetEntity() any {
	return c.entity
}

func (c *SchemaAwareVariableContext) GetField(path string) (any, error) {
	schemaDef, err := c.schemaRegistry.GetSchema(c.entityType)
	if err != nil {
		return nil, fmt.Errorf("schema not found: %s", c.entityType)
	}

	fieldSource, ok := schemaDef.FieldSources[path]
	if !ok {
		for name, source := range schemaDef.FieldSources {
			if strings.EqualFold(name, path) {
				fieldSource = source
				ok = true
				break
			}
		}

		if !ok {
			for _, source := range schemaDef.FieldSources {
				if source.Path == path {
					fieldSource = source
					ok = true
					break
				}
			}
		}

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

	return c.resolver.ResolveField(c.entity, fieldSource)
}

func (c *SchemaAwareVariableContext) GetComputed(function string) (any, error) {
	return c.resolver.ResolveComputed(c.entity, &schema.FieldSource{
		Computed: true,
		Function: function,
	})
}

func (c *SchemaAwareVariableContext) GetMetadata() map[string]any {
	return map[string]any{
		"entityType": c.entityType,
		"entity":     c.entity,
	}
}

func (c *SchemaAwareVariableContext) GetFieldSources() map[string]any {
	result := make(map[string]any)

	schemaDef, err := c.schemaRegistry.GetSchema(c.entityType)
	if err != nil {
		return result
	}

	for name := range schemaDef.FieldSources {
		if value, err := c.GetField(name); err == nil {
			result[name] = value
		}
	}

	return result
}

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
