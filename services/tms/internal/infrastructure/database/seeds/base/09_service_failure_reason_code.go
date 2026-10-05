package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

type ServiceFailureReasonCodeSeed struct {
	seedhelpers.BaseSeed
}

func NewServiceFailureReasonCodeSeed() *ServiceFailureReasonCodeSeed {
	seed := &ServiceFailureReasonCodeSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"ServiceFailureReasonCode",
		"1.0.0",
		"Creates default service failure reason codes for tenant operations",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)

	seed.SetDependencies(seedhelpers.SeedAdminAccount)

	return seed
}

func (s *ServiceFailureReasonCodeSeed) Run(ctx context.Context, tx bun.Tx) error {
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

			var createdCount int
			var createdOrgCount int
			for i := range orgs {
				count, err := tenantbootstrap.CreateServiceFailureReasonCodes(ctx, tx, tenantbootstrap.Scope{
					OrganizationID: orgs[i].ID,
					BusinessUnitID: orgs[i].BusinessUnitID,
					Record:         seedRecorder(sc, s.Name()),
				})
				if err != nil {
					return fmt.Errorf(
						"create default service failure reason codes for org %s: %w",
						orgs[i].Name,
						err,
					)
				}
				if count == 0 {
					continue
				}
				createdCount += count
				createdOrgCount++
			}

			if createdCount > 0 {
				seedhelpers.LogSuccess(
					"Created service failure reason code fixtures",
					fmt.Sprintf("- Created defaults for %d organizations", createdOrgCount),
					fmt.Sprintf("- Created %d service failure reason codes", createdCount),
				)
			}

			return nil
		},
	)
}

func (s *ServiceFailureReasonCodeSeed) Down(ctx context.Context, tx bun.Tx) error {
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

func (s *ServiceFailureReasonCodeSeed) CanRollback() bool {
	return true
}
