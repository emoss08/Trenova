package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/docling"
	"go.uber.org/fx"
)

var DoclingModule = fx.Module("docling",
	docling.Module,
)
