package jobs

import (
	"github.com/emoss08/trenova/internal/modules/accounting/jobs/accountingsyncjobs"
	"github.com/emoss08/trenova/internal/modules/accounting/jobs/fiscaljobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		accountingsyncjobs.Module,
		fiscaljobs.Module,
	)
}
