package formula

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/environment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/rules"
)

const (
	computedVariableCategory = "computed"
	providedVariableCategory = "context"
	itemFieldSeparator       = "[]."
)

func (s *Service) DescribeSchema(
	schemaID string,
) (*formulatemplatetypes.SchemaDescription, error) {
	variables, err := s.SchemaVariables(schemaID)
	if err != nil {
		return nil, err
	}

	return &formulatemplatetypes.SchemaDescription{
		SchemaID:  schemaID,
		Variables: variables,
	}, nil
}

func (s *Service) SchemaVariables(schemaID string) ([]rules.FormulaVariable, error) {
	variables, ok := s.schemas[schemaID]
	if !ok {
		return nil, errortypes.NewValidationError(
			"schemaId",
			errortypes.ErrInvalid,
			"Unknown formula schema: {0}", schemaID,
		)
	}
	return variables, nil
}

func schemaVariables(definition *formulatypes.Definition, schemaID string) []rules.FormulaVariable {
	rootCategory := definition.FormulaContext.Category
	if rootCategory == "" {
		rootCategory = schemaID
	}

	variables := collectSchemaVariables(definition.Properties, "", rootCategory, false)
	slices.SortFunc(variables, func(a, b rules.FormulaVariable) int {
		if a.Category != b.Category {
			return strings.Compare(a.Category, b.Category)
		}
		return strings.Compare(a.Name, b.Name)
	})
	return variables
}

func collectSchemaVariables(
	properties map[string]formulatypes.Property,
	prefix, category string,
	optionalParent bool,
) []rules.FormulaVariable {
	variables := make([]rules.FormulaVariable, 0, len(properties))

	for name := range properties {
		prop := properties[name]

		path := name
		if prefix != "" && strings.HasSuffix(prefix, itemFieldSeparator) {
			path = prefix + name
		} else if prefix != "" {
			path = prefix + "." + name
		}

		variableType, nullable := environment.TypeOf(&prop)

		if len(prop.Properties) > 0 {
			nestedCategory := name
			if prefix != "" {
				nestedCategory = category
			}
			variables = append(
				variables,
				collectSchemaVariables(
					prop.Properties,
					path,
					nestedCategory,
					optionalParent || nullable,
				)...)
			continue
		}

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

		variables = append(variables, rules.FormulaVariable{
			Name:        path,
			Type:        variableType,
			Description: prop.Description,
			Category:    variableCategory,
			Nullable:    nullable || optionalParent,
			Computed:    prop.Source.Computed,
			Enum:        prop.Enum,
		})

		if variableType == "array" && prop.Items != nil && len(prop.Items.Properties) > 0 {
			variables = append(
				variables,
				itemVariables(prop.Items.Properties, path+itemFieldSeparator, variableCategory,
					prop.Source.Computed)...,
			)
		}
	}

	return variables
}

func itemVariables(
	properties map[string]formulatypes.Property,
	prefix, category string,
	computed bool,
) []rules.FormulaVariable {
	items := collectSchemaVariables(properties, prefix, category, false)
	for index := range items {
		items[index].Category = category
		items[index].Computed = items[index].Computed || computed
	}
	return items
}
