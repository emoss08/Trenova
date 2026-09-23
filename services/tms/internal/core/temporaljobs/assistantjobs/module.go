package assistantjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"go.uber.org/fx"
)

var Module = fx.Module("assistant-jobs",
	fx.Provide(NewActivities, NewWorkflows, newArtifactObserver, NewTurnCanceller),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
