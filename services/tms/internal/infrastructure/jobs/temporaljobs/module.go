package temporaljobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/jobs/temporaljobs/jobscheduler"
	"github.com/emoss08/trenova/internal/infrastructure/jobs/temporaljobs/notificationjobs"
	"github.com/emoss08/trenova/internal/infrastructure/jobs/temporaljobs/shipmentjobs"
	"github.com/emoss08/trenova/internal/infrastructure/jobs/temporaljobs/systemjobs"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"temporal-jobs",
	ActivitiesModule,
	ClientModule,
	WorkerModule,
	SchedulerModule,
)

var ActivitiesModule = fx.Module(
	"temporal-activities",
	fx.Provide(
		shipmentjobs.NewActivities,
		notificationjobs.NewActivities,
		systemjobs.NewActivities,
	),
)

var ClientModule = fx.Module(
	"temporal-client",
	fx.Provide(
		NewTemporalClient,
	),
)

var WorkerModule = fx.Module(
	"temporal-worker",
	fx.Invoke(
		shipmentjobs.NewWorker,
		notificationjobs.NewWorker,
		systemjobs.NewWorker,
	),
)

var SchedulerModule = fx.Module(
	"temporal-scheduler",
	fx.Invoke(
		jobscheduler.NewScheduler,
	),
)
