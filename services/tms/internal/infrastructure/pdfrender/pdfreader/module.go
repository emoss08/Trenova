package pdfreader

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"pdf-reader",
	fx.Provide(
		fx.Annotate(New, fx.As(new(services.PDFReader))),
	),
)
