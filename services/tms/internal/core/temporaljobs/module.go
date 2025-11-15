package temporaljobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/auditjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/emailjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/jobscheduler"
	"github.com/emoss08/trenova/internal/core/temporaljobs/notificationjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/reportjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/searchjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/shipmentjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/workflowjobs"
	"go.uber.org/fx"
)

var Module = fx.Module("temporaljobs",
	ClientModule,
	WorkerModule,
	auditjobs.Module,
	notificationjobs.Module,
	emailjobs.Module,
	reportjobs.Module,
	searchjobs.Module,
	shipmentjobs.Module,
	workflowjobs.Module,
	SchedulerModule,
)

var ClientModule = fx.Module("temporal-client",
	fx.Provide(NewTemporalClient),
)

var WorkerModule = fx.Module("temporal-workers",
	fx.Provide(registry.NewWorkerManager),
)

var SchedulerModule = fx.Module("temporal-scheduler",
	fx.Provide(
		jobscheduler.NewManager,
	),
	fx.Invoke(jobscheduler.NewScheduler),
)
