package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/reporttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolResultReporter = (*reporttools.CreateReportTool)(nil)

var _ agentports.ToolResultReporter = (*reporttools.ForkReportTool)(nil)
