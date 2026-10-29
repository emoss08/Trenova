package workforcefixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/dotrandom"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/dotrandom/dotrandomdb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const (
	seededPoolCode           = "DOT-ALL"
	seededDrugRatePercent    = 50
	seededAlcoholRatePercent = 10
)

type DOTRandomPoolSeed struct {
}

func (s *DOTRandomPoolSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	pool := &dotrandom.DOTRandomPool{
		OrganizationID:      org.ID,
		BusinessUnitID:      org.BusinessUnitID,
		Code:                seededPoolCode,
		Name:                "All DOT drivers",
		Description:         "Every CDL driver in safety-sensitive work, drawn each quarter.",
		Status:              domaintypes.StatusActive,
		Period:              dotrandom.RandomPeriodQuarterly,
		DrugRatePercent:     seededDrugRatePercent,
		AlcoholRatePercent:  seededAlcoholRatePercent,
		IsDefault:           true,
		IncludedDriverTypes: []string{},
	}
	if err = dotrandomdb.DOTRandomPools.Create(ctx, db, pool); err != nil {
		return fmt.Errorf("insert the random testing pool: %w", err)
	}

	return nil
}
