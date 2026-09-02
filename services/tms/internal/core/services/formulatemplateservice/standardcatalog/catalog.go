// Package standardcatalog is the single source of truth for the standard
// rating templates every organization gets: the development seeder and the
// production install endpoint both read this embedded catalog, so the two can
// never drift apart.
package standardcatalog

import (
	_ "embed"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"gopkg.in/yaml.v3"
)

//go:embed standard_templates.yaml
var catalogYAML []byte

type Template struct {
	Name                string
	Description         string
	Type                formulatemplate.TemplateType
	Expression          string
	SchemaID            string
	VariableDefinitions []*formulatypes.VariableDefinition
}

type rawVariable struct {
	Name         string   `yaml:"name"`
	Type         string   `yaml:"type"`
	Description  string   `yaml:"description"`
	Required     bool     `yaml:"required"`
	DefaultValue *float64 `yaml:"default_value"`
}

type rawTemplate struct {
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Type        string        `yaml:"type"`
	Expression  string        `yaml:"expression"`
	SchemaID    string        `yaml:"schema_id"`
	Variables   []rawVariable `yaml:"variables"`
}

type rawCatalog struct {
	Templates []rawTemplate `yaml:"templates"`
}

func Load() ([]Template, error) {
	var raw rawCatalog
	if err := yaml.Unmarshal(catalogYAML, &raw); err != nil {
		return nil, fmt.Errorf("parse standard template catalog: %w", err)
	}

	templates := make([]Template, 0, len(raw.Templates))
	for _, entry := range raw.Templates {
		variables := make([]*formulatypes.VariableDefinition, 0, len(entry.Variables))
		for _, variable := range entry.Variables {
			definition := &formulatypes.VariableDefinition{
				Name:        variable.Name,
				Type:        variableType(variable.Type),
				Description: variable.Description,
				Required:    variable.Required,
			}
			if variable.DefaultValue != nil {
				definition.DefaultValue = *variable.DefaultValue
			}
			variables = append(variables, definition)
		}

		schemaID := entry.SchemaID
		if schemaID == "" {
			schemaID = "shipment"
		}

		templates = append(templates, Template{
			Name:                entry.Name,
			Description:         entry.Description,
			Type:                templateType(entry.Type),
			Expression:          entry.Expression,
			SchemaID:            schemaID,
			VariableDefinitions: variables,
		})
	}

	return templates, nil
}

func templateType(value string) formulatemplate.TemplateType {
	switch value {
	case "accessorial_charge":
		return formulatemplate.TemplateTypeAccessorialCharge
	default:
		return formulatemplate.TemplateTypeFreightCharge
	}
}

func variableType(value string) formulatypes.VariableValueType {
	switch value {
	case "string":
		return formulatypes.VariableValueTypeString
	case "boolean":
		return formulatypes.VariableValueTypeBoolean
	default:
		return formulatypes.VariableValueTypeNumber
	}
}
