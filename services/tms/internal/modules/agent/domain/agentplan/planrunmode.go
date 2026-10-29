package agentplan

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/shared/pulid"
)

type PlanRunMode string

const (
	PlanRunModeInOrder  = PlanRunMode("InOrder")
	PlanRunModeEachStep = PlanRunMode("EachStep")
)

func (m PlanRunMode) IsValid() bool {
	switch m {
	case PlanRunModeInOrder, PlanRunModeEachStep:
		return true
	default:
		return false
	}
}

func (m PlanRunMode) String() string { return string(m) }

func PlanRunModeOf(steps []*agent.AgentProposal) PlanRunMode {
	if len(steps) < 2 || steps[0] == nil {
		return PlanRunModeInOrder
	}

	resource := steps[0].TargetResource
	seen := make(map[pulid.ID]struct{}, len(steps))
	for _, step := range steps {
		if step == nil || step.TargetID.IsNil() || step.TargetResource == "" ||
			step.TargetResource != resource {
			return PlanRunModeInOrder
		}
		if _, repeated := seen[step.TargetID]; repeated {
			return PlanRunModeInOrder
		}
		seen[step.TargetID] = struct{}{}
	}

	return PlanRunModeEachStep
}
