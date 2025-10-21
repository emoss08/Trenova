package temporaljobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/ailogjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/auditjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/emailjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/jobscheduler"
	"github.com/emoss08/trenova/internal/core/temporaljobs/notificationjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/shipmentjobs"
	"go.uber.org/fx"
)

var Module = fx.Module("temporaljobs",
	ClientModule,
	WorkerModule,
	auditjobs.Module,
	notificationjobs.Module,
	emailjobs.Module,
	ailogjobs.Module,
	shipmentjobs.Module,
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
