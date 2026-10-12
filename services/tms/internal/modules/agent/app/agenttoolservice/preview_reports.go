package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/reporttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*reporttools.CreateReportTool)(nil)

var _ agentports.ToolPreviewer = (*reporttools.UpdateReportTool)(nil)

var _ agentports.ToolPreviewer = (*reporttools.ForkReportTool)(nil)
