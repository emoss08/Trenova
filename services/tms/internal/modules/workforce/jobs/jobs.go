package jobs

import (
	"github.com/emoss08/trenova/internal/modules/workforce/jobs/telematicsjobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		telematicsjobs.Module,
	)
}
