package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/alerttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*alerttools.AddHomeWidgetTool)(nil)

var _ agentports.ToolPreviewer = (*alerttools.RemoveHomeWidgetTool)(nil)

var _ agentports.ToolPreviewer = (*alerttools.ArrangeHomeLayoutTool)(nil)
