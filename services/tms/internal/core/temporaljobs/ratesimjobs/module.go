package ratesimjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"go.uber.org/fx"
)

// Module registers the simulation worker.
//
// There is no schedule provider: a simulation runs because somebody asked for
// one, not on a clock.
var Module = fx.Module("rate-simulation-jobs",
	fx.Provide(NewActivities),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
