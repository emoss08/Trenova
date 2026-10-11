package jobs

import (
	"github.com/emoss08/trenova/internal/modules/rating/jobs/formulatemplatejobs"
	"github.com/emoss08/trenova/internal/modules/rating/jobs/fuelpricejobs"
	"github.com/emoss08/trenova/internal/modules/rating/jobs/ratesimjobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		fuelpricejobs.Module,
		ratesimjobs.Module,
		formulatemplatejobs.Module,
	)
}
