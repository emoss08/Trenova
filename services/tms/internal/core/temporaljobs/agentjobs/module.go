package agentjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"go.uber.org/fx"
)

// asRegistry wires one of the agent's registries into the worker group. Which
// of them a given process actually runs is decided by --queues at startup, not
// here: every worker binary knows about all three and polls only the queues it
// was told to.
func asRegistry(constructor any) any {
	return fx.Annotate(
		constructor,
		fx.As(new(registry.WorkerRegistry)),
		fx.ResultTags(`group:"worker_registries"`),
	)
}

var Module = fx.Module("agent-jobs",
	fx.Provide(NewActivities),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
	fx.Provide(
		asRegistry(NewBackgroundRegistry),
		asRegistry(NewHeavyRegistry),
		asRegistry(NewDrainRegistry),
	),
)
