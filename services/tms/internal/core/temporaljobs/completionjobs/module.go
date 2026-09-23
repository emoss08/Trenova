package completionjobs

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"go.uber.org/fx"
)

var Module = fx.Module("completion-jobs",
	fx.Provide(
		NewActivities,
		NewDispatcher,
		func(d *Dispatcher) serviceports.StructuredCompleter { return d },
		func(d *Dispatcher) serviceports.AIProviderTester { return d },
		func(d *Dispatcher) serviceports.BriefingDayWriter { return d },
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
