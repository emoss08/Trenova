package portaljobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"go.uber.org/fx"
)

var Module = fx.Module("agent-portal-jobs",
	fx.Provide(NewActivities),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
