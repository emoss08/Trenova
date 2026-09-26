package accountingsyncjobs

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"go.uber.org/fx"
)

var Module = fx.Module("accounting-sync-jobs",
	fx.Provide(NewActivities),
	fx.Provide(fx.Annotate(
		NewReferenceRefresher,
		fx.As(new(services.AccountingReferenceRefresher)),
	)),
	fx.Provide(fx.Annotate(
		NewSyncDispatcher,
		fx.As(new(services.AccountingSyncDispatcher)),
	)),
	fx.Provide(fx.Annotate(
		NewChangesPoller,
		fx.As(new(services.AccountingChangePoller)),
	)),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
