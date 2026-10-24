package reportengine

import (
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"go.uber.org/fx"
)

var Module = fx.Module("report-engine",
	fx.Provide(
		NewCatalog,
		fx.Annotate(New, fx.As(new(reportports.ReportEngine))),
	),
)
