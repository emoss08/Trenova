package jobs

import (
	"github.com/emoss08/trenova/internal/modules/settlement/jobs/carriersettlementjobs"
	"github.com/emoss08/trenova/internal/modules/settlement/jobs/iftajobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		carriersettlementjobs.Module,
		iftajobs.Module,
	)
}
