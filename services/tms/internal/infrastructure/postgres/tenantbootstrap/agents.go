package tenantbootstrap

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	SystemAgentKeyBillingException   = "billing_exception"
	SystemAgentKeyDispatchAssignment = "dispatch_assignment"
)

func CreateSystemAgentDefinitions(ctx context.Context, db *keel.DB, scope Scope) (int, error) {
	orgID, buID := scope.OrganizationID, scope.BusinessUnitID
	d := agentdefinitiondb.Definitions
	existing, err := scoped(ctx, d.Pick(d.SystemKey).Where(
		d.OrganizationID.Eq(orgID),
		d.BusinessUnitID.Eq(buID),
		d.SystemKey.IsNotNull(),
	)).All(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("list system agents for organization %s: %w", orgID, err)
	}

	present := make(map[string]struct{}, len(existing))
	for _, row := range existing {
		present[row.SystemKey] = struct{}{}
	}

	created := 0
	now := scope.now()
	for _, definition := range SystemAgentDefinitions(orgID, buID) {
		if _, ok := present[definition.SystemKey]; ok {
			continue
		}

		definition.ApplyDefaults()
		if definition.TriggerMode == agentdefinition.TriggerScheduled {
			next, nextErr := definition.ComputeNextRun(now)
			if nextErr != nil {
				return created, fmt.Errorf(
					"schedule system agent %s: %w",
					definition.SystemKey,
					nextErr,
				)
			}
			definition.NextRunAt = &next
		}

		if err = d.Create(ctx, db, definition); err != nil {
			return created, fmt.Errorf("insert system agent %s: %w", definition.SystemKey, err)
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
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           dispatch.Label(),
			Description:    dispatch.Description(),
			Template:       dispatch,
			Icon:           agentdefinition.IconRoute,
			Accent:         agentdefinition.AccentIndigo,
			Instructions:   dispatch.StarterInstructions(),
			ToolNames:      dispatch.StarterTools(),
			ToolTiers: map[string]agent.AutonomyTier{
				"dispatch_assign_move": agent.TierPropose,
			},
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
