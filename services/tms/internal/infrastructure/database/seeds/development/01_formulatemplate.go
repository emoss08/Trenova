package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice/standardcatalog"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/buncolgen/formulatemplatecol"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type FormulaTemplateSeed struct {
	seedhelpers.BaseSeed
}

func NewFormulaTemplateSeed() *FormulaTemplateSeed {
	seed := &FormulaTemplateSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"FormulaTemplate",
		"2.0.0",
		"Creates the standard rating method formula templates, approved, in Tariff",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	seed.SetDependencies(seedhelpers.SeedTestOrganizations, seedhelpers.SeedAdminAccount)

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

			admin, err := sc.GetUserByUsername(ctx, "admin")
			if err != nil {
				return fmt.Errorf("get admin user: %w", err)
			}

			catalog, err := standardcatalog.Load()
			if err != nil {
				return fmt.Errorf("load standard template catalog: %w", err)
			}

			var existing []formulatemplate.FormulaTemplate
			cols := formulatemplatecol.FormulaTemplateColumns
			if err := tx.NewSelect().
				Model(&existing).
				Column(cols.Name.String()).
				Where(cols.OrganizationID.Eq(), org.ID).
				Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
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

				checked, err := checkSeedSource(
					ctx,
					rules.TariffKindTariff,
					entry.Name,
					entry.Source,
					nil,
				)
				if err != nil {
					return err
				}

				now := timeutils.NowUnix()
				tmpl := formulatemplate.Seed{
					OrganizationID: org.ID,
					BusinessUnitID: org.BusinessUnitID,
					Name:           entry.Name,
					Description:    entry.Description,
					Source:         entry.Source,
					Inputs:         tarifftypes.InputDeclsFrom(checked.check.Inputs),
					TestCount:      checked.testCount,
					Status:         formulatemplate.StatusActive,
				}.Build()
				tmpl.ID = pulid.MustNew("ft_")
				tmpl.ApprovedByID = &admin.ID
				tmpl.ApprovedAt = &now

				if _, err = tx.NewInsert().Model(tmpl).Exec(ctx); err != nil {
					return fmt.Errorf("insert formula template %s: %w", tmpl.Name, err)
				}

				version := formulatemplate.NewVersionFromTemplate(
					tmpl, 1, admin.ID, "Standard template installed", nil,
				)
				version.EffectiveFrom = &now
				if _, err = tx.NewInsert().Model(version).Exec(ctx); err != nil {
					return fmt.Errorf("insert formula template version %s: %w", tmpl.Name, err)
				}
				if err = sc.TrackCreated(ctx, "formula_templates", tmpl.ID, s.Name()); err != nil {
					return fmt.Errorf("track formula template: %w", err)
				}
				if err = sc.TrackCreated(ctx, "formula_template_versions", version.ID, s.Name()); err != nil {
					return fmt.Errorf("track formula template version: %w", err)
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
