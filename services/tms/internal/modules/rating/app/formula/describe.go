package formula

import (
	"sort"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/engine"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
)

const (
	computedVariableCategory = "computed"
	providedVariableCategory = "context"
)

func (s *Service) DescribeSchema(
	schemaID string,
) (*formulatemplatetypes.SchemaDescription, error) {
	definition, ok := s.registry.Get(schemaID)
	if !ok {
		return nil, errortypes.NewValidationError(
			"schemaId",
			errortypes.ErrInvalid,
			"Unknown formula schema: {0}", schemaID,
		)
	}

	rootCategory := definition.FormulaContext.Category
	if rootCategory == "" {
		rootCategory = schemaID
	}

	variables := collectSchemaVariables(definition.Properties, "", rootCategory, false)
	sort.Slice(variables, func(i, j int) bool {
		if variables[i].Category != variables[j].Category {
			return variables[i].Category < variables[j].Category
		}
		return variables[i].Name < variables[j].Name
	})

	return &formulatemplatetypes.SchemaDescription{
		SchemaID:  schemaID,
		Variables: variables,
		Functions: describeFunctions(),
		Builtins:  describeBuiltins(),
		Disabled:  describeDisabled(),
	}, nil
}

func describeFunctions() []formulatemplatetypes.SchemaFunctionInfo {
	specs := engine.FunctionSpecs()
	functions := make([]formulatemplatetypes.SchemaFunctionInfo, 0, len(specs))
	for i := range specs {
		spec := &specs[i]
		info := functionInfo(spec.Name, spec.Signature, spec.FunctionShape)
		info.Description = spec.Description
		info.Example = spec.Example
		info.Category = spec.Category
		info.Operator = spec.Operator
		functions = append(functions, info)
	}
	return functions
}

func describeBuiltins() []formulatemplatetypes.SchemaFunctionInfo {
	specs := engine.BuiltinSpecs()
	builtins := make([]formulatemplatetypes.SchemaFunctionInfo, 0, len(specs))
	for _, spec := range specs {
		builtins = append(builtins, functionInfo(spec.Name, spec.Signature, spec.FunctionShape))
	}
	return builtins
}

func describeDisabled() []formulatemplatetypes.SchemaDisabledFunction {
	disabled := engine.DisabledFunctions()
	described := make([]formulatemplatetypes.SchemaDisabledFunction, 0, len(disabled))
	for _, fn := range disabled {
		described = append(described, formulatemplatetypes.SchemaDisabledFunction{
			Name:        fn.Name,
			Reason:      fn.Reason,
			Replacement: fn.Replacement,
		})
	}
	return described
}

func functionInfo(
	name, signature string,
	fnShape engine.FunctionShape,
) formulatemplatetypes.SchemaFunctionInfo {
	return formulatemplatetypes.SchemaFunctionInfo{
		Name:        name,
		Signature:   signature,
		MinArgs:     fnShape.MinArgs,
		MaxArgs:     fnShape.MaxArgs,
		Returns:     fnShape.Returns,
		NumericArgs: fnShape.NumericArgs,
		Predicate:   fnShape.Predicate,
	}
}

func collectSchemaVariables(
	properties map[string]formulatypes.Property,
	prefix, category string,
	optionalParent bool,
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
			_, objectNullable := normalizePropertyType(prop.Type)
			variables = append(
				variables,
				collectSchemaVariables(
					prop.Properties,
					path,
					nestedCategory,
					optionalParent || objectNullable || prop.Source.Nullable,
				)...)
			continue
		}

		variableType, nullable := normalizePropertyType(prop.Type)
		if prop.IsDateTime() {
			variableType = "datetime"
		}

		variableCategory := category
		if prop.Source.Computed && prefix == "" {
			variableCategory = computedVariableCategory
		}
		if prop.Source.Provided {
			variableCategory = providedVariableCategory
		}

		variables = append(variables, formulatemplatetypes.SchemaVariableInfo{
			Name:        path,
			Type:        variableType,
			Description: prop.Description,
			Category:    variableCategory,
			Nullable:    nullable || prop.Source.Nullable || optionalParent,
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
