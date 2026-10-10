package jobs

import (
	"github.com/emoss08/trenova/internal/modules/rating/jobs/formulatemplatejobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		formulatemplatejobs.Module,
	)
}
