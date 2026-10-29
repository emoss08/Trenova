package dispatchquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
)

var (
	weatherSeverityLevels = agenttoolschema.Source("weather.severity", weatherSeverities)
)
