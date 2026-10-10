package accountingsyncjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"go.uber.org/fx"
)

var Module = fx.Module("accounting-sync-jobs",
	fx.Provide(NewActivities),
	fx.Provide(fx.Annotate(
		NewReferenceRefresher,
		fx.As(new(accountingsyncports.AccountingReferenceRefresher)),
	)),
	fx.Provide(fx.Annotate(
		NewChangesPoller,
		fx.As(new(accountingsyncports.AccountingChangePoller)),
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
