package tenantbootstrap

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const (
	SystemAgentKeyBillingException   = "billing_exception"
	SystemAgentKeyDispatchAssignment = "dispatch_assignment"
)

func CreateSystemAgentDefinitions(ctx context.Context, tx bun.IDB, scope Scope) (int, error) {
	orgID, buID := scope.OrganizationID, scope.BusinessUnitID
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
	now := scope.now()
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
		if err := scope.record(ctx, "agent_definitions", definition.ID); err != nil {
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
				agentdefinition.ContextMemory,
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
				agentdefinition.ContextMemory,
			},
			Enabled: false,
		},
	}
}
