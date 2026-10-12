package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolSelectionResolver = (*operationstools.EvaluateServiceFailuresTool)(nil)
