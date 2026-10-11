package jobs

import (
	"github.com/emoss08/trenova/internal/modules/parties/jobs/carrierintelligencejobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		carrierintelligencejobs.Module,
	)
}
