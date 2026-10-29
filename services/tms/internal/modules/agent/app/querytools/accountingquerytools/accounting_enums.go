package accountingquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
)

var (
	accountingDriftSeverities = agenttoolschema.Source(
		"accountingDrift.severity",
		[]string{"Critical", "Warning", "Watch"},
	)
)
