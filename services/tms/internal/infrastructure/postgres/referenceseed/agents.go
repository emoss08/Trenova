package referenceseed

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentinstructions"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmark"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	SystemAgentKeyBillingException   = "billing_exception"
	SystemAgentKeyDispatchAssignment = "dispatch_assignment"
)

func CreateSystemAgentDefinitions(
	ctx context.Context,
	db *keel.DB,
	scope tenantscope.Scope,
) (int, error) {
	orgID, buID := scope.OrganizationID, scope.BusinessUnitID
	d := agentdefinitiondb.Definitions
	existing, err := tenantscope.Scoped(ctx, d.Pick(d.SystemKey).Where(
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
	now := scope.NowUnix()
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
	billing := agenttemplate.TemplateBillingException
	dispatch := agenttemplate.TemplateDispatchAssignment

	return []*agentdefinition.Definition{
		{
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Name:            billing.Label(),
			Description:     billing.Description(),
			Template:        billing,
			Icon:            agentmark.IconReceipt,
			Accent:          agentmark.AccentAmber,
			Instructions:    agentinstructions.StarterInstructions(billing),
			ToolNames:       agentpreset.StarterTools(billing),
			AutonomyCeiling: agentpreset.StarterCeiling(billing),
			TriggerMode:     agentpreset.StarterTrigger(billing),
			EventKinds:      agentpreset.StarterEvents(billing),
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
			Icon:           agentmark.IconRoute,
			Accent:         agentmark.AccentIndigo,
			Instructions:   agentinstructions.StarterInstructions(dispatch),
			ToolNames:      agentpreset.StarterTools(dispatch),
			ToolTiers: map[string]agent.AutonomyTier{
				"dispatch_assign_move": agent.TierPropose,
			},
			AutonomyCeiling: agentpreset.StarterCeiling(dispatch),
			TriggerMode:     agentpreset.StarterTrigger(dispatch),
			EventKinds:      agentpreset.StarterEvents(dispatch),
			CronExpression:  agentpreset.StarterCron(dispatch),
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
