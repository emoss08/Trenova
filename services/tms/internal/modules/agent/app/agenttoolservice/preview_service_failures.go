package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/servicefailuretools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*servicefailuretools.EvaluateServiceFailuresTool)(nil)
