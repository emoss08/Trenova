package oversightquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
)

var (
	watchtowerKindSource = agenttoolschema.Source(
		"watchtower.sourceKind",
		watchtowerKindValues(),
	)
	watchtowerSeveritySource = agenttoolschema.Source("watchtower.severity", []string{
		watchtower.SeverityCritical.String(),
		watchtower.SeverityWarning.String(),
		watchtower.SeverityInfo.String(),
	})
)
