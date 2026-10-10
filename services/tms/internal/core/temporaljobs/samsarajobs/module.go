package samsarajobs

import (
	"github.com/emoss08/trenova/internal/core/services/samsarasyncservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/samsarajobs/samsaracontract"
	"go.uber.org/fx"
)

var Module = fx.Module("samsara-jobs",
	fx.Provide(NewActivities),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
