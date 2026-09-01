package formula

import (
	"sort"

	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
)

const computedVariableCategory = "computed"

func (s *Service) DescribeSchema(
	schemaID string,
) (*formulatemplatetypes.SchemaDescription, error) {
	definition, ok := s.registry.Get(schemaID)
	if !ok {
		return nil, errortypes.NewValidationError(
			"schemaId",
			errortypes.ErrInvalid,
			"Unknown formula schema: "+schemaID,
		)
	}

	rootCategory := definition.FormulaContext.Category
	if rootCategory == "" {
		rootCategory = schemaID
	}

	variables := collectSchemaVariables(definition.Properties, "", rootCategory)
	sort.Slice(variables, func(i, j int) bool {
		if variables[i].Category != variables[j].Category {
			return variables[i].Category < variables[j].Category
		}
		return variables[i].Name < variables[j].Name
	})

	specs := engine.FunctionSpecs()
	functions := make([]formulatemplatetypes.SchemaFunctionInfo, 0, len(specs))
	for _, spec := range specs {
		functions = append(functions, formulatemplatetypes.SchemaFunctionInfo{
			Name:        spec.Name,
			Signature:   spec.Signature,
			Description: spec.Description,
			Example:     spec.Example,
			Category:    spec.Category,
			Operator:    spec.Operator,
		})
	}

	return &formulatemplatetypes.SchemaDescription{
		SchemaID:  schemaID,
		Variables: variables,
		Functions: functions,
	}, nil
}

func collectSchemaVariables(
	properties map[string]formulatypes.Property,
	prefix, category string,
) []formulatemplatetypes.SchemaVariableInfo {
	variables := make([]formulatemplatetypes.SchemaVariableInfo, 0, len(properties))

	for name := range properties {
		prop := properties[name]

		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		if len(prop.Properties) > 0 {
			nestedCategory := name
			if prefix != "" {
				nestedCategory = category
			}
			variables = append(
				variables,
				collectSchemaVariables(prop.Properties, path, nestedCategory)...)
			continue
		}

		variableType, nullable := normalizePropertyType(prop.Type)

		variableCategory := category
		if prop.Source.Computed && prefix == "" {
			variableCategory = computedVariableCategory
		}

		variables = append(variables, formulatemplatetypes.SchemaVariableInfo{
			Name:        path,
			Type:        variableType,
			Description: prop.Description,
			Category:    variableCategory,
			Nullable:    nullable || prop.Source.Nullable,
			Computed:    prop.Source.Computed,
			Enum:        prop.Enum,
		})
	}

	return variables
}

func normalizePropertyType(propType any) (string, bool) {
	switch typed := propType.(type) {
	case string:
		return typed, false
	case []any:
		nullable := false
		resolved := ""
		for _, entry := range typed {
			value, ok := entry.(string)
			if !ok {
				continue
			}
			if value == "null" {
				nullable = true
				continue
			}
			if resolved == "" {
				resolved = value
			}
		}
		return resolved, nullable
	default:
		return "", false
	}
}
