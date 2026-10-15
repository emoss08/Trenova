package jobs

import (
	"github.com/emoss08/trenova/internal/modules/identity/jobs/accessjobs"
	"github.com/emoss08/trenova/internal/modules/identity/jobs/billingjobs"
	"github.com/emoss08/trenova/internal/modules/identity/jobs/smsjobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		accessjobs.Module,
		billingjobs.Module,
		smsjobs.Module,
	)
}
