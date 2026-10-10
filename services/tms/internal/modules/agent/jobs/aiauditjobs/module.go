package aiauditjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"go.uber.org/fx"
)

var Module = fx.Module("ai-audit-jobs",
	fx.Provide(NewActivities, NewExportActivities),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
		fx.Annotate(
			NewExportRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
)
