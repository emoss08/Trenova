package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/reporttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*reporttools.CreateReportTool)(nil)

var _ agenttoolports.ToolPreviewer = (*reporttools.UpdateReportTool)(nil)

var _ agenttoolports.ToolPreviewer = (*reporttools.ForkReportTool)(nil)
