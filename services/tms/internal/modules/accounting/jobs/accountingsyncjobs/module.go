package accountingsyncjobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountinginboundports"
	"go.uber.org/fx"
)

var Module = fx.Module("accounting-sync-jobs",
	fx.Provide(NewActivities),
	fx.Provide(fx.Annotate(
		NewReferenceRefresher,
		fx.As(new(accountingconnectionports.AccountingReferenceRefresher)),
	)),
	fx.Provide(fx.Annotate(
		NewChangesPoller,
		fx.As(new(accountinginboundports.AccountingChangePoller)),
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
