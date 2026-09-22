package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	SystemAgentKeyBillingException   = "billing_exception"
	SystemAgentKeyDispatchAssignment = "dispatch_assignment"
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
				n, err := s.ensureForOrganization(
					ctx, tx, sc, organizations[i].ID, organizations[i].BusinessUnitID,
				)
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

func (s *SystemAgentDefinitionsSeed) ensureForOrganization(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) (int, error) {
	cols := buncolgen.DefinitionColumns
	var existing []string
	if err := tx.NewSelect().
		Model((*agentdefinition.Definition)(nil)).
		Column(cols.SystemKey.Name).
		Where(cols.OrganizationID.Eq(), orgID).
		Where(cols.BusinessUnitID.Eq(), buID).
		Where(cols.SystemKey.IsNotNull()).
		Scan(ctx, &existing); err != nil {
		return 0, fmt.Errorf("list system agents for organization %s: %w", orgID, err)
	}

	present := make(map[string]struct{}, len(existing))
	for _, key := range existing {
		present[key] = struct{}{}
	}

	created := 0
	now := timeutils.NowUnix()
	for _, definition := range SystemAgentDefinitions(orgID, buID) {
		if _, ok := present[definition.SystemKey]; ok {
			continue
		}

		definition.ApplyDefaults()
		if definition.TriggerMode == agentdefinition.TriggerScheduled {
			next, err := definition.ComputeNextRun(now)
			if err != nil {
				return created, fmt.Errorf("schedule system agent %s: %w", definition.SystemKey, err)
			}
			definition.NextRunAt = &next
		}

		if _, err := tx.NewInsert().Model(definition).Exec(ctx); err != nil {
			return created, fmt.Errorf("insert system agent %s: %w", definition.SystemKey, err)
		}
		if err := sc.TrackCreated(ctx, "agent_definitions", definition.ID, s.Name()); err != nil {
			return created, err
		}
		created++
	}

	return created, nil
}

func SystemAgentDefinitions(orgID, buID pulid.ID) []*agentdefinition.Definition {
	billing := agentdefinition.TemplateBillingException
	dispatch := agentdefinition.TemplateDispatchAssignment

	return []*agentdefinition.Definition{
		{
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Name:            billing.Label(),
			Description:     billing.Description(),
			Template:        billing,
			Icon:            agentdefinition.IconReceipt,
			Accent:          agentdefinition.AccentAmber,
			Instructions:    billing.StarterInstructions(),
			ToolNames:       billing.StarterTools(),
			AutonomyCeiling: billing.StarterCeiling(),
			TriggerMode:     billing.StarterTrigger(),
			EventKinds:      billing.StarterEvents(),
			OutputMode:      agentdefinition.OutputReport,
			ShadowMode:      true,
			SystemKey:       SystemAgentKeyBillingException,
			ContextProviders: []agentdefinition.ContextProvider{
				agentdefinition.ContextOrganization,
				agentdefinition.ContextClock,
				agentdefinition.ContextTools,
			},
			Enabled: false,
		},
		{
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Name:            dispatch.Label(),
			Description:     dispatch.Description(),
			Template:        dispatch,
			Icon:            agentdefinition.IconRoute,
			Accent:          agentdefinition.AccentIndigo,
			Instructions:    dispatch.StarterInstructions(),
			ToolNames:       dispatch.StarterTools(),
			ToolTiers:       map[string]agent.AutonomyTier{"assign_move": agent.TierPropose},
			AutonomyCeiling: dispatch.StarterCeiling(),
			TriggerMode:     dispatch.StarterTrigger(),
			EventKinds:      dispatch.StarterEvents(),
			CronExpression:  dispatch.StarterCron(),
			OutputMode:      agentdefinition.OutputReport,
			ShadowMode:      true,
			SystemKey:       SystemAgentKeyDispatchAssignment,
			ContextProviders: []agentdefinition.ContextProvider{
				agentdefinition.ContextOrganization,
				agentdefinition.ContextClock,
				agentdefinition.ContextTools,
			},
			Enabled: false,
		},
	}
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
