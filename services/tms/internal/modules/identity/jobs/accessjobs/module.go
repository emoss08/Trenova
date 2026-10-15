package accessjobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"go.uber.org/fx"
)

var Module = fx.Module("identity-access-jobs",
	fx.Provide(
		func(s *userservice.Service) Sweeper { return s },
		NewActivities,
	),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
