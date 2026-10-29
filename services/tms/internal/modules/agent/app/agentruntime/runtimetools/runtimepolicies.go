package runtimetools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/askuser"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

type runtimePolicySpec struct {
	name      string
	operation permission.Operation
	scope     agenttooltrust.ToolScope
	egress    agent.EgressClass
	effect    agenttooltrust.ToolEffect
	rationale string
}

var runtimePolicySpecs = []runtimePolicySpec{
	{
		name:      FindToolsName,
		operation: permission.OpRead,
		scope:     agenttooltrust.ToolScopeRun,
		egress:    agent.EgressNone,
		effect:    agenttooltrust.ToolEffectDiscover,
		rationale: "Loads more of the agent's own tools into the turn; it reads the " +
			"catalog and changes nothing.",
	},
	{
		name:      askuser.Name,
		operation: permission.OpRead,
		scope:     agenttooltrust.ToolScopeRun,
		egress:    agent.EgressNone,
		effect:    agenttooltrust.ToolEffectAsk,
		rationale: "Asks the person in the conversation a question; nothing is saved " +
			"or sent.",
	},
	{
		name:      PublishArtifactName,
		operation: permission.OpCreate,
		scope:     agenttooltrust.ToolScopeRun,
		egress:    agent.EgressPersonal,
		effect:    agenttooltrust.ToolEffectPresent,
		rationale: "Publishes a document into the caller's own conversation, where " +
			"only they read it.",
	},
	{
		name:      ShowBoardName,
		operation: permission.OpRead,
		scope:     agenttooltrust.ToolScopeRun,
		egress:    agent.EgressPersonal,
		effect:    agenttooltrust.ToolEffectPresent,
		rationale: "Lays out an answer as a board in the caller's own conversation. It writes " +
			"no business record, and every value on it is read again with the viewer's " +
			"access when the board is opened.",
	},
	{
		name:      RequestDecisionName,
		operation: permission.OpRead,
		scope:     agenttooltrust.ToolScopeRun,
		egress:    agent.EgressPersonal,
		effect:    agenttooltrust.ToolEffectAsk,
		rationale: "Opens, in the person's own conversation, the approval box on a " +
			"proposal already waiting on them; it decides nothing and changes no record.",
	},
	{
		name:      WithdrawProposalName,
		operation: permission.OpRead,
		scope:     agenttooltrust.ToolScopeRun,
		egress:    agent.EgressNone,
		effect:    agenttooltrust.ToolEffectAsk,
		rationale: "Takes back a proposal the agent filed earlier in the same turn, before " +
			"anyone has seen it; it changes no record and only narrows what the person is asked.",
	},
	{
		name:      NamePlanName,
		operation: permission.OpRead,
		scope:     agenttooltrust.ToolScopeRun,
		egress:    agent.EgressNone,
		effect:    agenttooltrust.ToolEffectAsk,
		rationale: "Names the plan the person approves the turn's proposals as. It writes no " +
			"business record: the name is kept only as the plan's own title, beside the " +
			"previewed changes the approval rests on.",
	},
	{
		name:      DelegateTaskName,
		operation: permission.OpCreate,
		scope:     agenttooltrust.ToolScopeRun,
		egress:    agent.EgressNone,
		effect:    agenttooltrust.ToolEffectDelegate,
		rationale: "Hands a task to another agent of the organization, which runs as " +
			"the same person under its own tiers.",
	},
}

func Policies() agenttoolports.RuntimeToolPolicies {
	policies := make(agenttoolports.RuntimeToolPolicies, 0, len(runtimePolicySpecs))
	for idx := range runtimePolicySpecs {
		policies = append(policies, runtimePolicySpecs[idx].policy())
	}

	return policies
}

func PolicyNamed(name string) (agenttoolports.ToolPolicy, bool) {
	for idx := range runtimePolicySpecs {
		if runtimePolicySpecs[idx].name == name {
			return runtimePolicySpecs[idx].policy(), true
		}
	}

	return agenttoolports.ToolPolicy{}, false
}

func (s *runtimePolicySpec) policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          s.name,
		Kind:          agenttooltrust.ToolKindRuntime,
		Resource:      permission.ResourceAssistant,
		Operation:     s.operation,
		Scope:         s.scope,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{s.egress},
		Effect:        s.effect,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     s.rationale,
	}
}
