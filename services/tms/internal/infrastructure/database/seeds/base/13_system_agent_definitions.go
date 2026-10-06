package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const (
	SystemAgentKeyBillingException   = tenantbootstrap.SystemAgentKeyBillingException
	SystemAgentKeyDispatchAssignment = tenantbootstrap.SystemAgentKeyDispatchAssignment
)

type SystemAgentDefinitionsSeed struct {
	seedhelpers.BaseSeed
}

func NewSystemAgentDefinitionsSeed() *SystemAgentDefinitionsSeed {
	seed := &SystemAgentDefinitionsSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"SystemAgentDefinitions",
		"1.0.0",
		"Creates the disabled, shadowed system agent definitions every organization can switch on",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)
	seed.SetDependencies(seedhelpers.SeedAdminAccount)

	return seed
}

func (s *SystemAgentDefinitionsSeed) Repeatable() bool {
	return true
}

func (s *SystemAgentDefinitionsSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			var organizations []tenant.Organization
			if err := tx.NewSelect().
				Model(&organizations).
				Column("id", "business_unit_id").
				Scan(ctx); err != nil {
				return fmt.Errorf("list organizations: %w", err)
			}

			created := 0
			for i := range organizations {
				n, err := tenantbootstrap.CreateSystemAgentDefinitions(ctx, tx, tenantbootstrap.Scope{
					OrganizationID: organizations[i].ID,
					BusinessUnitID: organizations[i].BusinessUnitID,
					Record:         seedRecorder(sc, s.Name()),
				})
				if err != nil {
					return err
				}
				created += n
			}

			if created > 0 {
				seedhelpers.LogSuccess(
					"Created system agent definitions",
					fmt.Sprintf("- Created %d system agent definitions", created),
				)
			}

			return nil
		},
	)
}

func SystemAgentDefinitions(orgID, buID pulid.ID) []*agentdefinition.Definition {
	return tenantbootstrap.SystemAgentDefinitions(orgID, buID)
}

func (s *SystemAgentDefinitionsSeed) Down(ctx context.Context, tx bun.Tx) error {
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

func (s *SystemAgentDefinitionsSeed) CanRollback() bool {
	return true
}
