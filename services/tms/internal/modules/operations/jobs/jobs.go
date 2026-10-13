package jobs

import (
	"github.com/emoss08/trenova/internal/modules/operations/jobs/checkcalljobs"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/detentionjobs"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/etajobs"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/recurringshipmentjobs"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/shipmentjobs"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/tenderjobs"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/weatheralertjobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		checkcalljobs.Module,
		detentionjobs.Module,
		etajobs.Module,
		shipmentjobs.Module,
		recurringshipmentjobs.Module,
		tenderjobs.Module,
		weatheralertjobs.Module,
	)
}
