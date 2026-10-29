package reportquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
)

var (
	reportRunFormats = agenttoolschema.Source("report.format", report.AllFormats())
)
