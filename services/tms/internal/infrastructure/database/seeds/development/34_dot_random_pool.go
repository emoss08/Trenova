package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

const (
	seededPoolCode           = "DOT-ALL"
	seededDrugRatePercent    = 50
	seededAlcoholRatePercent = 10
)

type DOTRandomPoolSeed struct {
	seedhelpers.BaseSeed
}

func NewDOTRandomPoolSeed() *DOTRandomPoolSeed {
	seed := &DOTRandomPoolSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"DOTRandomPool",
		"1.0.0",
		"Seeds the default DOT random testing pool at the FMCSA minimum rates",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedTestOrganizations)

	return seed
}

func (s *DOTRandomPoolSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}

			cols := buncolgen.DOTRandomPoolColumns
			exists, err := tx.NewSelect().
				Model((*worker.DOTRandomPool)(nil)).
				Where(cols.OrganizationID.Eq(), org.ID).
				Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
				Where(cols.Code.Eq(), seededPoolCode).
				Exists(ctx)
			if err != nil {
				return fmt.Errorf("look for the random testing pool: %w", err)
			}
			if exists {
				return nil
			}

			pool := &worker.DOTRandomPool{
				OrganizationID:     org.ID,
				BusinessUnitID:     org.BusinessUnitID,
				Code:               seededPoolCode,
				Name:               "All DOT drivers",
				Description:        "Every CDL driver in safety-sensitive work, drawn each quarter.",
				Status:             domaintypes.StatusActive,
				Period:             worker.RandomPeriodQuarterly,
				DrugRatePercent:    seededDrugRatePercent,
				AlcoholRatePercent: seededAlcoholRatePercent,
				IsDefault:          true,
			}
			if _, err = tx.NewInsert().Model(pool).Exec(ctx); err != nil {
				return fmt.Errorf("insert the random testing pool: %w", err)
			}

			return nil
		},
	)
}
