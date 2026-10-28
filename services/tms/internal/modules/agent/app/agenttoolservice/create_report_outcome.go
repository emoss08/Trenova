package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/reporttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolResultReporter = (*reporttools.CreateReportTool)(nil)

var _ agenttoolports.ToolResultReporter = (*reporttools.ForkReportTool)(nil)
