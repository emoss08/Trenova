package jobs

import (
	"github.com/emoss08/trenova/internal/modules/places/jobs/distancemileagejobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		distancemileagejobs.Module,
	)
}
