package agentdefinitionresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func saveRequestFromInput(
	id pulid.ID,
	input *gqlmodel.SaveAgentDefinitionInput,
	tenantInfo pagination.TenantInfo,
) (*agentdefinitionports.SaveAgentDefinitionRequest, error) {
	toolTiers, err := toolTiersFromInput(input.ToolTiers)
	if err != nil {
		return nil, err
	}
	toolDailyLimits, err := toolDailyLimitsFromInput(input.ToolDailyLimits)
	if err != nil {
		return nil, err
	}
	budget, err := gqldecimal.ParseNullDecimalField("monthlyBudgetUsd", input.MonthlyBudgetUsd)
	if err != nil {
		return nil, err
	}
	preferredProviderID, err := gqlid.ParseOptionalIDField(
		"preferredProviderId",
		input.PreferredProviderID,
	)
	if err != nil {
		return nil, err
	}
	access, err := accessFromInput(input)
	if err != nil {
		return nil, err
	}

	req := &agentdefinitionports.SaveAgentDefinitionRequest{
		ID:                     id,
		Name:                   input.Name,
		Description:            input.Description,
		Instructions:           input.Instructions,
		Guardrails:             input.Guardrails,
		ToolNames:              input.ToolNames,
		ToolTiers:              toolTiers,
		AutonomyCeiling:        input.AutonomyCeiling,
		DataAccessCeiling:      input.DataAccessCeiling,
		Enabled:                input.Enabled,
		ShadowMode:             input.ShadowMode,
		DecisionTimeoutSeconds: input.DecisionTimeoutSeconds,
		Icon:                   input.Icon,
		Accent:                 input.Accent,
		TriggerMode:            input.TriggerMode,
		CronExpression:         input.CronExpression,
		CronTimezone:           input.CronTimezone,
		EventKinds:             eventKindsFromInput(input.EventKinds),
		IntervalSeconds:        input.IntervalSeconds,
		EndsAt:                 gqlscalar.Int64Ptr(input.EndsAt),
		MaxConcurrentRuns:      input.MaxConcurrentRuns,
		RunTimeoutSeconds:      input.RunTimeoutSeconds,
		MaxToolCalls:           input.MaxToolCalls,
		MonthlyBudgetUSD:       gqldecimal.NullDecimalValue(budget),
		DailyRunLimit:          input.DailyRunLimit,
		ToolDailyLimits:        toolDailyLimits,
		SimulationMode:         input.SimulationMode,
		MemoryTokenBudget:      input.MemoryTokenBudget,
		LearningOff:            input.LearningOff,
		ContextProviders:       input.ContextProviders,
		OutputMode:             input.OutputMode,
		PreferredProviderID:    preferredProviderID,
		Access:                 access,
		Version:                int64(input.Version),
		TenantInfo:             tenantInfo,
	}
	if input.Template != nil {
		req.Template = *input.Template
	}
	if input.DelegateIds != nil {
		delegates, parseErr := gqlid.ParseIDListField("delegateIds", input.DelegateIds)
		if parseErr != nil {
			return nil, parseErr
		}
		req.DelegateIDs = &delegates
	}

	return req, nil
}

func toolTiersFromInput(
	tiers []*gqlmodel.AgentToolTierInput,
) (map[string]agent.AutonomyTier, error) {
	out := make(map[string]agent.AutonomyTier, len(tiers))
	for _, tier := range tiers {
		if _, seen := out[tier.Tool]; seen {
			return nil, errortypes.NewValidationError(
				"toolTiers", errortypes.ErrDuplicate, "Each tool may be given one tier",
			)
		}
		out[tier.Tool] = tier.Tier
	}

	return out, nil
}

func toolDailyLimitsFromInput(limits []*gqlmodel.AgentToolDailyLimitInput) (map[string]int, error) {
	out := make(map[string]int, len(limits))
	for _, limit := range limits {
		if _, seen := out[limit.Tool]; seen {
			return nil, errortypes.NewValidationError(
				"toolDailyLimits", errortypes.ErrDuplicate, "Each tool may be given one limit",
			)
		}
		out[limit.Tool] = limit.Limit
	}

	return out, nil
}

func eventKindsFromInput(kinds []string) []agenttrigger.EventKind {
	out := make([]agenttrigger.EventKind, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, agenttrigger.EventKind(kind))
	}

	return out
}

func accessFromInput(
	input *gqlmodel.SaveAgentDefinitionInput,
) (*agentdefinitionports.AgentAccessWrite, error) {
	switch {
	case input.AccessMode == nil && input.AccessRoleIds == nil:
		return nil, nil //nolint:nilnil // no access in the input keeps who may use the agent
	case input.AccessMode == nil:
		return nil, errortypes.NewValidationError(
			"accessMode", errortypes.ErrRequired,
			"Access is required when roles are given",
		)
	case input.AccessRoleIds == nil:
		return nil, errortypes.NewValidationError(
			"accessRoleIds", errortypes.ErrRequired,
			"Roles are required when access is given; send an empty list for none",
		)
	}

	roleIDs, err := gqlid.ParseIDListField("accessRoleIds", input.AccessRoleIds)
	if err != nil {
		return nil, err
	}

	return &agentdefinitionports.AgentAccessWrite{Mode: *input.AccessMode, RoleIDs: roleIDs}, nil
}
