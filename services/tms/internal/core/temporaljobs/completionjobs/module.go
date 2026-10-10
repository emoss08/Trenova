package completionjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"go.uber.org/fx"
)

var Module = fx.Module("completion-jobs",
	fx.Provide(
		NewActivities,
		NewDispatcher,
		func(d *Dispatcher) agentports.StructuredCompleter { return d },
		func(d *Dispatcher) agentports.AIProviderTester { return d },
		func(d *Dispatcher) briefingports.BriefingDayWriter { return d },
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
