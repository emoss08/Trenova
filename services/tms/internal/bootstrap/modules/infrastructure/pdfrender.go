package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/captureqr"
	"github.com/emoss08/trenova/internal/infrastructure/imaging"
	"github.com/emoss08/trenova/internal/infrastructure/pdfrender/gotenberg"
	"go.uber.org/fx"
)

var PDFRenderModule = fx.Module("pdf-render",
	gotenberg.Module,
	imaging.Module,
	fx.Provide(
		captureqr.New,
	),
)
