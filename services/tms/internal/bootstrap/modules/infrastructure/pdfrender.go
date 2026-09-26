package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/captureimaging"
	"github.com/emoss08/trenova/internal/infrastructure/pdfassembly"
	"github.com/emoss08/trenova/internal/infrastructure/pdfrender/gotenberg"
	"go.uber.org/fx"
)

var PDFRenderModule = fx.Module("pdf-render",
	gotenberg.Module,
	fx.Provide(pdfassembly.New, captureimaging.New),
)
