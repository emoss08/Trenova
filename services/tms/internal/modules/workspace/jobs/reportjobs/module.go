package reportjobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/workspace/jobs/reportcatalogjobs"
	"go.uber.org/fx"
)

var Module = fx.Module("report-jobs",
	fx.Provide(NewActivities, reportcatalogjobs.NewActivities),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
	fx.Invoke(reportcatalogjobs.StartCatalogHeartbeat),
)
