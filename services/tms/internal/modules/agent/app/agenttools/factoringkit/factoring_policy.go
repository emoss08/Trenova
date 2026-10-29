package factoringkit

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const (
	FieldFactoringCompany = "factoringCompanyId"
	ParamVersion          = "version"
	ParamFactorID         = "factoringCompanyId"
	ParamReason           = "reason"
)

type PolicySpec struct {
	Resource    permission.Resource
	Operation   permission.Operation
	DefaultTier agent.AutonomyTier
	MaxTier     agent.AutonomyTier
	Egress      agent.EgressClass
	Idempotent  bool
	Reversible  bool
	Rationale   string
}

func Policy(name string, spec *PolicySpec) agenttoolports.ToolPolicy {
	egress := spec.Egress
	if egress == "" {
		egress = agent.EgressInternal
	}
	return agenttoolports.ToolPolicy{
		Name:          name,
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      spec.Resource,
		Operation:     spec.Operation,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   spec.DefaultTier,
		MaxTier:       spec.MaxTier,
		Egress:        []agent.EgressClass{egress},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    spec.Reversible,
		Idempotent:    spec.Idempotent,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     spec.Rationale,
	}
}

func Refused(summary string, err error) (*agentpreview.ToolPreview, error) {
	if changepreview.IsRefusal(err) {
		return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
	}
	return nil, err
}

const MaxVersion = 1 << 30
