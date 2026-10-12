package jobs

import (
	"github.com/emoss08/trenova/internal/modules/reference/jobs/exchangeratejobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		exchangeratejobs.Module,
	)
}
