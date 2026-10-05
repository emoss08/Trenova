package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/captureimaging"
	"github.com/emoss08/trenova/internal/infrastructure/captureqr"
	"github.com/emoss08/trenova/internal/infrastructure/officedoc"
	"github.com/emoss08/trenova/internal/infrastructure/pdfassembly"
	"github.com/emoss08/trenova/internal/infrastructure/pdfrender/gotenberg"
	"github.com/emoss08/trenova/internal/infrastructure/pdfrender/pdfreader"
	"go.uber.org/fx"
)

var PDFRenderModule = fx.Module("pdf-render",
	gotenberg.Module,
	pdfreader.Module,
	fx.Provide(pdfassembly.New, captureimaging.New, captureqr.New, officedoc.New),
)
