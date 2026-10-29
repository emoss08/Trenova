package pricingfixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice/standardcatalog"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func RunFormulaTemplates(ctx context.Context, r *seed.Run) error {
	org, err := tenantfixtures.DefaultOrganization(ctx, r)
	if err != nil {
		return err
	}

	admin, err := tenantfixtures.AdminUser(ctx, r.DB)
	if err != nil {
		return err
	}

	catalog, err := standardcatalog.Load()
	if err != nil {
		return fmt.Errorf("load standard template catalog: %w", err)
	}

	for _, entry := range catalog {
		checked, err := checkSeedSource(ctx, rules.TariffKindTariff, entry.Name, entry.Source, nil)
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

		if err = formulatemplatedb.FormulaTemplates.Create(ctx, r.DB, tmpl); err != nil {
			return fmt.Errorf("insert formula template %s: %w", tmpl.Name, err)
		}

		version := formulatemplate.NewVersionFromTemplate(
			tmpl, 1, admin.ID, "Standard template installed", nil,
		)
		version.EffectiveFrom = &now
		if err = formulatemplatedb.FormulaTemplateVersions.Create(ctx, r.DB, version); err != nil {
			return fmt.Errorf("insert formula template version %s: %w", tmpl.Name, err)
		}
		r.Count(1)
	}

	return nil
}
