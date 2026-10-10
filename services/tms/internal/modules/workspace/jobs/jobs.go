package jobs

import (
	"github.com/emoss08/trenova/internal/modules/workspace/app/bulkeditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/jobs/auditjobs"
	"github.com/emoss08/trenova/internal/modules/workspace/jobs/bulkeditjobs"
	"github.com/emoss08/trenova/internal/modules/workspace/jobs/emailjobs"
	"github.com/emoss08/trenova/internal/modules/workspace/jobs/fuelcardjobs"
	"github.com/emoss08/trenova/internal/modules/workspace/jobs/reportjobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			func(svc *bulkeditservice.Service) bulkeditjobs.Runner { return svc },
		),
		auditjobs.Module,
		bulkeditjobs.Module,
		emailjobs.Module,
		fuelcardjobs.Module,
		reportjobs.Module,
	)
}
