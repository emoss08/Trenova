package pdfreader

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"pdf-reader",
	fx.Provide(
		fx.Annotate(New, fx.As(new(kernelports.PDFReader))),
	),
)
