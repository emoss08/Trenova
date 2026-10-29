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
	resource   permission.Resource
	operation  permission.Operation
	tier       agent.AutonomyTier
	reversible bool
	idempotent bool
	rationale  string
}

func SyncPolicy(name string, spec SyncPolicySpec) agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          name,
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      spec.resource,
		Operation:     spec.operation,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   spec.tier,
		MaxTier:       spec.tier,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    spec.reversible,
		Idempotent:    spec.idempotent,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     spec.rationale,
	}
}

func RefusedSync(summary string, err error) (*agentpreview.ToolPreview, error) {
	if changepreview.IsRefusal(err) {
		return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
	}

	return nil, err
}
