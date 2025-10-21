package base

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

// PermissionsAndPoliciesSeed Creates permissions and policies data
type PermissionsAndPoliciesSeed struct {
	seedhelpers.BaseSeed
}

// NewPermissionsAndPoliciesSeed creates a new permissions_and_policies seed
func NewPermissionsAndPoliciesSeed() *PermissionsAndPoliciesSeed {
	seed := &PermissionsAndPoliciesSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"PermissionsAndPolicies",
		"1.0.0",
		"Creates permissions and policies data",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)

	return seed
}

// Run executes the seed
func (s *PermissionsAndPoliciesSeed) Run(ctx context.Context, db *bun.DB) error {
	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
			// This seed is intentionally empty as policies and roles are created
			// in the admin_account seed after the business unit and organization are created

			seedhelpers.LogSuccess("Permissions and policies fixtures",
				"- Policies and roles are created in admin_account seed",
			)

			return nil
		},
	)
}
