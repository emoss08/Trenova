package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice/standardcatalog"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
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
		"1.2.0",
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

			catalog, err := standardcatalog.Load()
			if err != nil {
				return fmt.Errorf("load standard template catalog: %w", err)
			}

			// A version bump re-runs the whole seed, so templates already
			// present are skipped by name: the bump only adds what the catalog
			// gained, and never duplicates the standard set an organization's
			// contracts already point at.
			var existing []formulatemplate.FormulaTemplate
			if err := tx.NewSelect().
				Model(&existing).
				Column("name").
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Scan(ctx); err != nil {
				return fmt.Errorf("load existing formula templates: %w", err)
			}

			existingNames := make(map[string]struct{}, len(existing))
			for _, template := range existing {
				existingNames[template.Name] = struct{}{}
			}

			for _, entry := range catalog {
				if _, ok := existingNames[entry.Name]; ok {
					continue
				}

				tmpl := &formulatemplate.FormulaTemplate{
					ID:                  pulid.MustNew("ft_"),
					OrganizationID:      org.ID,
					BusinessUnitID:      org.BusinessUnitID,
					Name:                entry.Name,
					Description:         entry.Description,
					Type:                entry.Type,
					Expression:          entry.Expression,
					Status:              formulatemplate.StatusActive,
					SchemaID:            entry.SchemaID,
					VariableDefinitions: entry.VariableDefinitions,
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
