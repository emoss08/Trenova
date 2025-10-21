package base

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazmatexpiration"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

// HazmatExpirationSeed Creates hazmat expiration data
type HazmatExpirationSeed struct {
	seedhelpers.BaseSeed
}

// NewHazmatExpirationSeed creates a new hazmat_expiration seed
func NewHazmatExpirationSeed() *HazmatExpirationSeed {
	seed := &HazmatExpirationSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"HazmatExpiration",
		"1.0.0",
		"Creates hazmat expiration data",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)

	return seed
}

// Run executes the seed
func (s *HazmatExpirationSeed) Run(ctx context.Context, db *bun.DB) error {
	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
			var count int
			count, err := db.NewSelect().
				Model((*hazmatexpiration.HazmatExpiration)(nil)).
				ColumnExpr("count(*)").
				Count(ctx)
			if err != nil {
				return err
			}

			if count > 0 {
				seedhelpers.LogSuccess("Hazmat expirations already exist, skipping")
				return nil
			}

			flID, err := seedCtx.GetStateByAbbreviation("FL")
			if err != nil {
				return err
			}

			hazmatExpiration := &hazmatexpiration.HazmatExpiration{
				StateID: flID,
				Years:   4,
			}

			if _, err := tx.NewInsert().Model(hazmatExpiration).Exec(ctx); err != nil {
				return err
			}

			seedhelpers.LogSuccess("Created hazmat expiration fixtures",
				"- 1 hazmat expiration created",
			)

			return nil
		},
	)
}
