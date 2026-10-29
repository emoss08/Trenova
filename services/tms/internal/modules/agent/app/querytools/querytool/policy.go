package querytool

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const readRationale = "Reads records the caller may already open; it changes nothing and " +
	"sends nothing."

type ReadSpec struct {
	Resource        permission.Resource
	Scope           agenttooltrust.ToolScope
	Effect          agenttooltrust.ToolEffect
	Reads           agenttooltrust.ExternalRead
	Source          agent.TaintSource
	CapsLaterWrites bool
	Rationale       string
}

func ReadPolicy(name string, spec *ReadSpec) agenttoolports.ToolPolicy {
	policy := agenttoolports.ToolPolicy{
		Name:            name,
		Kind:            agenttooltrust.ToolKindQuery,
		Resource:        spec.Resource,
		Operation:       permission.OpRead,
		Scope:           spec.Scope,
		DefaultTier:     agent.TierAutoExecute,
		MaxTier:         agent.TierAutoExecute,
		Egress:          []agent.EgressClass{agent.EgressNone},
		Effect:          spec.Effect,
		Reversible:      true,
		Idempotent:      true,
		ReadsExternal:   spec.Reads,
		Source:          spec.Source,
		CapsLaterWrites: spec.CapsLaterWrites,
		Rationale:       spec.Rationale,
	}
	if policy.Scope == "" {
		policy.Scope = agenttooltrust.ToolScopeTenant
	}
	if policy.Effect == "" {
		policy.Effect = agenttooltrust.ToolEffectLookup
	}
	if policy.ReadsExternal == "" {
		policy.ReadsExternal = agenttooltrust.ExternalReadNever
	}
	if policy.Rationale == "" {
		policy.Rationale = readRationale
	}

	return policy
}
