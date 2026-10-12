package jobs

import (
	"github.com/emoss08/trenova/internal/modules/edi/jobs/edijobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		edijobs.Module,
	)
}
