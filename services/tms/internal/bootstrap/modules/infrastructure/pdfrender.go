package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/pdfrender/gotenberg"
	"go.uber.org/fx"
)

var PDFRenderModule = fx.Module("pdf-render", gotenberg.Module)
