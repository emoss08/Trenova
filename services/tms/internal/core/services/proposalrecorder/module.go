package proposalrecorder

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"go.uber.org/fx"
)

var Module = fx.Module("proposalrecorder",
	fx.Provide(New),
	fx.Provide(NewPlanStore),
)

func NewPlanStore(plans agentports.AgentPlanRepository) PlanStore {
	return plans
}
