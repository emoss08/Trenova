package proposalrecorder

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"go.uber.org/fx"
)

var Module = fx.Module("proposalrecorder",
	fx.Provide(New),
	fx.Provide(NewPlanStore),
	fx.Provide(NewScriptSteps),
)

func NewPlanStore(plans agentplanports.AgentPlanRepository) PlanStore {
	return plans
}

func NewScriptSteps(steps agentruntimeports.AgentRunStepRepository) ScriptStepReader {
	return steps
}
