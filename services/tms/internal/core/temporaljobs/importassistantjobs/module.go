package importassistantjobs

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"go.uber.org/fx"
)

var Module = fx.Module("import-assistant-jobs",
	fx.Provide(
		NewActivities,
		NewTurns,
		func(t *Turns) serviceports.ShipmentImportTurns { return t },
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
