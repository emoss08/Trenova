package accountingtools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

type SyncPolicySpec struct {
	Resource   permission.Resource
	Operation  permission.Operation
	Tier       agent.AutonomyTier
	Reversible bool
	Idempotent bool
	Rationale  string
}

func SyncPolicy(name string, spec SyncPolicySpec) agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          name,
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      spec.Resource,
		Operation:     spec.Operation,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   spec.Tier,
		MaxTier:       spec.Tier,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    spec.Reversible,
		Idempotent:    spec.Idempotent,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     spec.Rationale,
	}
}

func RefusedSync(summary string, err error) (*agentpreview.ToolPreview, error) {
	if changepreview.IsRefusal(err) {
		return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
	}

	return nil, err
}
