package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/dashboardtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*dashboardtools.SaveTableViewTool)(nil)
