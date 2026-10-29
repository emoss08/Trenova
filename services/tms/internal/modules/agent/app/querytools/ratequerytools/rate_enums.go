package ratequerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
)

var (
	ratingSides = agenttoolschema.Source(
		"rating.side",
		[]string{"Customer", "Carrier"},
	)
)
