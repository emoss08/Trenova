package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const (
	seededPoolCode           = "DOT-ALL"
	seededDrugRatePercent    = 50
	seededAlcoholRatePercent = 10
)

type DOTRandomPoolSeed struct {
}

var DOTRandomPool = Seeds.Add(seed.Seed{
	Name:        "dot_random_pool",
	Kind:        seed.Fixture,
	Description: "Seeds the default DOT random testing pool at the FMCSA minimum rates",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&DOTRandomPoolSeed{}).Run,
})

func (s *DOTRandomPoolSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	pool := &worker.DOTRandomPool{
		OrganizationID:      org.ID,
		BusinessUnitID:      org.BusinessUnitID,
		Code:                seededPoolCode,
		Name:                "All DOT drivers",
		Description:         "Every CDL driver in safety-sensitive work, drawn each quarter.",
		Status:              domaintypes.StatusActive,
		Period:              worker.RandomPeriodQuarterly,
		DrugRatePercent:     seededDrugRatePercent,
		AlcoholRatePercent:  seededAlcoholRatePercent,
		IsDefault:           true,
		IncludedDriverTypes: []string{},
	}
	if err = workerdb.DOTRandomPools.Create(ctx, db, pool); err != nil {
		return fmt.Errorf("insert the random testing pool: %w", err)
	}

	return nil
}
