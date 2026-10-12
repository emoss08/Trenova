package jobs

import (
	"github.com/emoss08/trenova/internal/modules/settlement/jobs/carriersettlementjobs"
	"github.com/emoss08/trenova/internal/modules/settlement/jobs/iftajobs"
	"github.com/emoss08/trenova/internal/modules/settlement/jobs/settlementjobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		settlementjobs.Module,
		carriersettlementjobs.Module,
		iftajobs.Module,
	)
}
