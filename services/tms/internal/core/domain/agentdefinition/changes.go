package agentdefinition

import (
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/editchange"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/setutils"
	"github.com/emoss08/trenova/shared/typeutils"
)

// changeRules lists, in the builder's order, every setting a person edits.
// Bookkeeping the server keeps on its own (when it last ran, who turned it
// off) is not a change anyone made, so it is left out.
var changeRules = []editchange.Rule[Definition]{
	{Field: "name", Label: "Name", Same: func(a, b *Definition) bool { return a.Name == b.Name }},
	{
		Field: "description", Label: "Description",
		Same: func(a, b *Definition) bool { return a.Description == b.Description },
	},
	{
		Field: "icon", Label: "Look",
		Same: func(a, b *Definition) bool { return a.Icon == b.Icon && a.Accent == b.Accent },
	},
	{
		Field: "instructions", Label: "Instructions",
		Same: func(a, b *Definition) bool { return a.Instructions == b.Instructions },
	},
	{
		Field: "guardrails", Label: "Never list",
		Same: func(a, b *Definition) bool { return slices.Equal(a.Guardrails, b.Guardrails) },
	},
	{Field: "triggerMode", Label: "When it runs", Same: sameSchedule},
	{
		Field: "accessMode", Label: "Who can use it",
		Same: func(a, b *Definition) bool { return a.AccessMode == b.AccessMode },
	},
	{
		Field: "toolNames", Label: "Tools",
		Same: func(a, b *Definition) bool {
			return setutils.SameMembers(a.ToolNames, b.ToolNames) &&
				setutils.SameMembers(a.DisabledToolNames, b.DisabledToolNames)
		},
	},
	{
		Field: "toolTiers", Label: "Freedom per tool",
		Same: func(a, b *Definition) bool { return maps.Equal(a.ToolTiers, b.ToolTiers) },
	},
	{
		Field: "toolDailyLimits", Label: "Daily limits",
		Same: func(a, b *Definition) bool { return maps.Equal(a.ToolDailyLimits, b.ToolDailyLimits) },
	},
	{
		Field: "autonomyCeiling", Label: "Ceiling",
		Same: func(a, b *Definition) bool { return a.AutonomyCeiling == b.AutonomyCeiling },
	},
	{
		Field: "dataAccessCeiling", Label: "Data access",
		Same: func(a, b *Definition) bool { return a.DataAccessCeiling == b.DataAccessCeiling },
	},
	{
		Field: "decisionTimeoutSeconds", Label: "Proposals expire after",
		Same: func(a, b *Definition) bool {
			return a.DecisionTimeoutSeconds == b.DecisionTimeoutSeconds
		},
	},
	{
		Field: "maxChangeItems", Label: "Records per change",
		Same: func(a, b *Definition) bool { return a.MaxChangeItems == b.MaxChangeItems },
	},
	{Field: "businessHoursOnly", Label: "Business hours", Same: sameBusinessHours},
	{
		Field: "monthlyBudgetUsd", Label: "Monthly budget",
		Same: func(a, b *Definition) bool {
			return decimalutils.PtrEqual(a.MonthlyBudgetUSD, b.MonthlyBudgetUSD)
		},
	},
	{
		Field: "dailyRunLimit", Label: "Runs per day",
		Same: func(a, b *Definition) bool { return a.DailyRunLimit == b.DailyRunLimit },
	},
	{
		Field: "runTimeoutSeconds", Label: "Run timeout",
		Same: func(a, b *Definition) bool { return a.RunTimeoutSeconds == b.RunTimeoutSeconds },
	},
	{
		Field: "maxToolCalls", Label: "Tool calls per run",
		Same: func(a, b *Definition) bool { return a.MaxToolCalls == b.MaxToolCalls },
	},
	{
		Field: "preferredProviderId", Label: "Preferred provider",
		Same: func(a, b *Definition) bool { return a.PreferredProviderID == b.PreferredProviderID },
	},
	{
		Field: "outputMode", Label: "Replies as",
		Same: func(a, b *Definition) bool { return a.OutputMode == b.OutputMode },
	},
	{
		Field: "learnsFromWork", Label: "Learns from its work",
		Same: func(a, b *Definition) bool { return a.LearningOff == b.LearningOff },
	},
	{
		Field: "memoryTokenBudget", Label: "Memory in the prompt",
		Same: func(a, b *Definition) bool {
			return typeutils.EqualPtr(a.MemoryTokenBudget, b.MemoryTokenBudget)
		},
	},
	{
		Field: "contextProviders", Label: "Tell it about",
		Same: func(a, b *Definition) bool {
			return setutils.SameMembers(a.ContextProviders, b.ContextProviders)
		},
	},
	{
		Field: "delegateIds", Label: "Can ask",
		Same: func(a, b *Definition) bool {
			return setutils.SameMembers(a.DelegateIDs, b.DelegateIDs) &&
				maps.Equal(a.DelegateTopics, b.DelegateTopics)
		},
	},
	{
		Field: "mode", Label: "Mode",
		Same: func(a, b *Definition) bool {
			return a.ShadowMode == b.ShadowMode && a.SimulationMode == b.SimulationMode
		},
	},
	{
		Field: "enabled", Label: "Enabled",
		Same: func(a, b *Definition) bool { return a.Enabled == b.Enabled },
	},
}

func sameSchedule(a, b *Definition) bool {
	return a.TriggerMode == b.TriggerMode &&
		a.CronExpression == b.CronExpression &&
		a.CronTimezone == b.CronTimezone &&
		setutils.SameMembers(a.EventKinds, b.EventKinds) &&
		a.IntervalSeconds == b.IntervalSeconds &&
		typeutils.EqualPtr(a.EndsAt, b.EndsAt) &&
		a.MaxConcurrentRuns == b.MaxConcurrentRuns
}

func sameBusinessHours(a, b *Definition) bool {
	return a.BusinessHoursOnly == b.BusinessHoursOnly &&
		a.BusinessHoursStart == b.BusinessHoursStart &&
		a.BusinessHoursEnd == b.BusinessHoursEnd &&
		a.BusinessHoursTimezone == b.BusinessHoursTimezone
}

// Changes lists the settings that differ between two saves of an agent, in
// the order the builder shows them. A nil before is a new agent, which has
// no changes of its own to list.
func Changes(before, after *Definition) []editchange.Change {
	return editchange.Detect(changeRules, before, after)
}
