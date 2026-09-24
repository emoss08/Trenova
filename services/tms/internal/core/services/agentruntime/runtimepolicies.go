package agentruntime

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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
		name:      delegateTaskName,
		operation: permission.OpCreate,
		scope:     agent.ToolScopeRun,
		egress:    agent.EgressNone,
		effect:    agent.ToolEffectDelegate,
		rationale: "Hands a task to another agent of the organization, which runs as " +
			"the same person under its own tiers.",
	},
}

func RuntimePolicies() serviceports.RuntimeToolPolicies {
	policies := make(serviceports.RuntimeToolPolicies, 0, len(runtimePolicySpecs))
	for idx := range runtimePolicySpecs {
		policies = append(policies, runtimePolicySpecs[idx].policy())
	}

	return policies
}

func runtimePolicyNamed(name string) (serviceports.ToolPolicy, bool) {
	for idx := range runtimePolicySpecs {
		if runtimePolicySpecs[idx].name == name {
			return runtimePolicySpecs[idx].policy(), true
		}
	}

	return serviceports.ToolPolicy{}, false
}

func (s runtimePolicySpec) policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
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
