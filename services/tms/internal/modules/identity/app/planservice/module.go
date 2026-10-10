package planservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"go.uber.org/fx"
)

var Module = fx.Module("plan-service", fx.Provide(New))

func New() platformplanports.PlanService {
	return NewUnlimited()
}
