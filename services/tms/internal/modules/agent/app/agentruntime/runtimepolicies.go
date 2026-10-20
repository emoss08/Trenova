package agentruntime

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

type runtimePolicySpec struct {
	name      string
	operation permission.Operation
	scope     agent.ToolScope
	egress    agent.EgressClass
	effect    agent.ToolEffect
	rationale string
}

var runtimePolicySpecs = []runtimePolicySpec{
	{
		name:      findToolsName,
		operation: permission.OpRead,
		scope:     agent.ToolScopeRun,
		egress:    agent.EgressNone,
		effect:    agent.ToolEffectDiscover,
		rationale: "Loads more of the agent's own tools into the turn; it reads the " +
			"catalog and changes nothing.",
	},
	{
		name:      askUserName,
		operation: permission.OpRead,
		scope:     agent.ToolScopeRun,
		egress:    agent.EgressNone,
		effect:    agent.ToolEffectAsk,
		rationale: "Asks the person in the conversation a question; nothing is saved " +
			"or sent.",
	},
	{
		name:      publishArtifactName,
		operation: permission.OpCreate,
		scope:     agent.ToolScopeRun,
		egress:    agent.EgressPersonal,
		effect:    agent.ToolEffectPresent,
		rationale: "Publishes a document into the caller's own conversation, where " +
			"only they read it.",
	},
	{
		name:      requestDecisionName,
		operation: permission.OpRead,
		scope:     agent.ToolScopeRun,
		egress:    agent.EgressPersonal,
		effect:    agent.ToolEffectAsk,
		rationale: "Opens, in the person's own conversation, the approval box on a " +
			"proposal already waiting on them; it decides nothing and changes no record.",
	},
	{
		name:      withdrawProposalName,
		operation: permission.OpRead,
		scope:     agent.ToolScopeRun,
		egress:    agent.EgressNone,
		effect:    agent.ToolEffectAsk,
		rationale: "Takes back a proposal the agent filed earlier in the same turn, before " +
			"anyone has seen it; it changes no record and only narrows what the person is asked.",
	},
	{
		name:      namePlanName,
		operation: permission.OpRead,
		scope:     agent.ToolScopeRun,
		egress:    agent.EgressNone,
		effect:    agent.ToolEffectAsk,
		rationale: "Names the plan the person approves the turn's proposals as. It writes no " +
			"business record: the name is kept only as the plan's own title, beside the " +
			"previewed changes the approval rests on.",
	},
	{
		name:      delegateTaskName,
		operation: permission.OpCreate,
		scope:     agent.ToolScopeRun,
		egress:    agent.EgressNone,
		effect:    agent.ToolEffectDelegate,
		rationale: "Hands a task to another agent of the organization, which runs as " +
			"the same person under its own tiers.",
	},
}

func RuntimePolicies() agentports.RuntimeToolPolicies {
	policies := make(agentports.RuntimeToolPolicies, 0, len(runtimePolicySpecs))
	for idx := range runtimePolicySpecs {
		policies = append(policies, runtimePolicySpecs[idx].policy())
	}

	return policies
}

func runtimePolicyNamed(name string) (agentports.ToolPolicy, bool) {
	for idx := range runtimePolicySpecs {
		if runtimePolicySpecs[idx].name == name {
			return runtimePolicySpecs[idx].policy(), true
		}
	}

	return agentports.ToolPolicy{}, false
}

func (s runtimePolicySpec) policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          s.name,
		Kind:          agent.ToolKindRuntime,
		Resource:      permission.ResourceAssistant,
		Operation:     s.operation,
		Scope:         s.scope,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{s.egress},
		Effect:        s.effect,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     s.rationale,
	}
}
