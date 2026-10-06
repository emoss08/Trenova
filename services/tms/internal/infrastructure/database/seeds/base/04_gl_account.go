package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

type GLAccountSeed struct {
	seedhelpers.BaseSeed
}

func NewGLAccountSeed() *GLAccountSeed {
	seed := &GLAccountSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"GLAccount",
		"1.0.0",
		"Creates default account types and chart of accounts for trucking operations",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)

	seed.SetDependencies(seedhelpers.SeedAdminAccount)

	return seed
}

func (s *GLAccountSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			orgs, err := listOrganizations(ctx, tx)
			if err != nil {
				return err
			}

			var createdOrgCount int
			var totalAccountCount int
			for i := range orgs {
				org := &orgs[i]

				exists, err := tenantbootstrap.HasChartOfAccounts(ctx, tx, org.ID, org.BusinessUnitID)
				if err != nil {
					return fmt.Errorf("check existing account types for org %s: %w", org.Name, err)
				}

				if exists {
					continue
				}

				accountCount, err := tenantbootstrap.CreateChartOfAccounts(ctx, tx, tenantbootstrap.Scope{
					OrganizationID: org.ID,
					BusinessUnitID: org.BusinessUnitID,
					Record:         seedRecorder(sc, s.Name()),
				})
				if err != nil {
					return fmt.Errorf("create default COA for org %s: %w", org.Name, err)
				}

				totalAccountCount += accountCount
				createdOrgCount++
			}

			if totalAccountCount > 0 {
				seedhelpers.LogSuccess(
					"Created GL account fixtures",
					fmt.Sprintf("- Created default account types for %d organizations", createdOrgCount),
					fmt.Sprintf("- Created %d GL accounts for trucking operations", totalAccountCount),
				)
			}

			return nil
		},
	)
}

func (s *GLAccountSeed) Down(ctx context.Context, tx bun.Tx) error {
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

func (s *GLAccountSeed) CanRollback() bool {
	return true
}
