package jobs

import (
	"github.com/emoss08/trenova/internal/modules/workforce/jobs/compliancejobs"
	"github.com/emoss08/trenova/internal/modules/workforce/jobs/ptojobs"
	"github.com/emoss08/trenova/internal/modules/workforce/jobs/samsarajobs"
	"github.com/emoss08/trenova/internal/modules/workforce/jobs/telematicsjobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		ptojobs.Module,
		samsarajobs.Module,
		compliancejobs.Module,
		telematicsjobs.Module,
	)
}
