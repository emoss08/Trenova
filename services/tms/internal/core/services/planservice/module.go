package planservice

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

var Module = fx.Module("plan-service", fx.Provide(New))

func New() services.PlanService {
	return NewUnlimited()
}
