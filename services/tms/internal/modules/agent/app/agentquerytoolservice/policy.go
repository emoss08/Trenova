package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

const readRationale = "Reads records the caller may already open; it changes nothing and " +
	"sends nothing."

type readSpec struct {
	resource  permission.Resource
	scope     agent.ToolScope
	effect    agent.ToolEffect
	reads     agent.ExternalRead
	source    agent.TaintSource
	rationale string
}

func readPolicy(name string, spec readSpec) agentports.ToolPolicy {
	policy := agentports.ToolPolicy{
		Name:          name,
		Kind:          agent.ToolKindQuery,
		Resource:      spec.resource,
		Operation:     permission.OpRead,
		Scope:         spec.scope,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressNone},
		Effect:        spec.effect,
		Reversible:    true,
		Idempotent:    true,
		ReadsExternal: spec.reads,
		Source:        spec.source,
		Rationale:     spec.rationale,
	}
	if policy.Scope == "" {
		policy.Scope = agent.ToolScopeTenant
	}
	if policy.Effect == "" {
		policy.Effect = agent.ToolEffectLookup
	}
	if policy.ReadsExternal == "" {
		policy.ReadsExternal = agent.ExternalReadNever
	}
	if policy.Rationale == "" {
		policy.Rationale = readRationale
	}

	return policy
}
