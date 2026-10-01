package proposalrecorder

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"go.uber.org/fx"
)

var Module = fx.Module("proposalrecorder",
	fx.Provide(New),
	fx.Provide(NewPlanStore),
)

func NewPlanStore(plans repositories.AgentPlanRepository) PlanStore {
	return plans
}
