package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*masterdatatools.CreateDashboardTool)(nil)

var _ agentports.ToolPreviewer = (*masterdatatools.AddDashboardTileTool)(nil)
