package jobs

import (
	"github.com/emoss08/trenova/internal/modules/documents/jobs/capturejobs"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionjobs"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentuploadjobs"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/inboundjobs"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/thumbnailjobs"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/triagewakejobs"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/triageworkjobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		documentextractionjobs.Module,
		documentuploadjobs.Module,
		thumbnailjobs.Module,
		inboundjobs.Module,
		capturejobs.Module,
		triagewakejobs.Module,
		triageworkjobs.Module,
	)
}
