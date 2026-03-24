package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type FormulaTemplateSeed struct {
	seedhelpers.BaseSeed
}

func NewFormulaTemplateSeed() *FormulaTemplateSeed {
	seed := &FormulaTemplateSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"FormulaTemplate",
		"1.0.0",
		"Creates standard rating method formula templates",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies(seedhelpers.SeedTestOrganizations)

	return seed
}

func (s *FormulaTemplateSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetOrganization("default_org")
			if err != nil {
				org, err = sc.GetDefaultOrganization(ctx)
				if err != nil {
					return fmt.Errorf("get organization: %w", err)
				}
			}

			loader := seedhelpers.NewDataLoader(
				"./internal/infrastructure/database/seeds/development/data",
			)

			var data struct {
				Templates []struct {
					Name        string `yaml:"name"`
					Description string `yaml:"description"`
					Type        string `yaml:"type"`
					Expression  string `yaml:"expression"`
					Status      string `yaml:"status"`
					SchemaID    string `yaml:"schema_id"`
					Variables   []struct {
						Name         string  `yaml:"name"`
						Type         string  `yaml:"type"`
						Description  string  `yaml:"description"`
						Required     bool    `yaml:"required"`
						DefaultValue float64 `yaml:"default_value"`
					} `yaml:"variables"`
				} `yaml:"templates"`
			}

			if err := loader.LoadYAML("formula_templates.yaml", &data); err != nil {
				return fmt.Errorf("load formula templates: %w", err)
			}

			for _, tmplData := range data.Templates {
				variableDefs := make([]*formulatypes.VariableDefinition, len(tmplData.Variables))
				for i, v := range tmplData.Variables {
					variableDefs[i] = &formulatypes.VariableDefinition{
						Name:         v.Name,
						Type:         stringToVariableType(v.Type),
						Description:  v.Description,
						Required:     v.Required,
						DefaultValue: v.DefaultValue,
					}
				}

				tmpl := &formulatemplate.FormulaTemplate{
					ID:                  pulid.MustNew("ft_"),
					OrganizationID:      org.ID,
					BusinessUnitID:      org.BusinessUnitID,
					Name:                tmplData.Name,
					Description:         tmplData.Description,
					Type:                stringToTemplateType(tmplData.Type),
					Expression:          tmplData.Expression,
					Status:              stringToStatus(tmplData.Status),
					SchemaID:            tmplData.SchemaID,
					VariableDefinitions: variableDefs,
				}

				if _, err := tx.NewInsert().Model(tmpl).Exec(ctx); err != nil {
					return fmt.Errorf("insert formula template %s: %w", tmpl.Name, err)
				}
				if err := sc.TrackCreated(ctx, "formula_templates", tmpl.ID, s.Name()); err != nil {
					return fmt.Errorf("track formula template: %w", err)
				}
			}

			return nil
		},
	)
}

func stringToTemplateType(s string) formulatemplate.TemplateType {
	switch s {
	case "freight_charge":
		return formulatemplate.TemplateTypeFreightCharge
	case "accessorial_charge":
		return formulatemplate.TemplateTypeAccessorialCharge
	default:
		return formulatemplate.TemplateTypeFreightCharge
	}
}

func stringToStatus(s string) formulatemplate.Status {
	switch s {
	case "active":
		return formulatemplate.StatusActive
	case "inactive":
		return formulatemplate.StatusInactive
	default:
		return formulatemplate.StatusActive
	}
}

func stringToVariableType(s string) formulatypes.VariableValueType {
	switch s {
	case "number":
		return formulatypes.VariableValueTypeNumber
	case "string":
		return formulatypes.VariableValueTypeString
	case "boolean":
		return formulatypes.VariableValueTypeBoolean
	default:
		return formulatypes.VariableValueTypeNumber
	}
}

func (s *FormulaTemplateSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *FormulaTemplateSeed) CanRollback() bool {
	return true
}
