package telematicsjobs

import (
	"github.com/emoss08/trenova/internal/core/services/telematicsservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"go.uber.org/fx"
)

var Module = fx.Module("telematics-jobs",
	fx.Provide(telematicsservice.New),
	fx.Provide(NewActivities),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
