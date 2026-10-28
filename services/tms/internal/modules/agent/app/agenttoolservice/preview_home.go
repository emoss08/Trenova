package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/alerttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*alerttools.AddHomeWidgetTool)(nil)

var _ agenttoolports.ToolPreviewer = (*alerttools.RemoveHomeWidgetTool)(nil)

var _ agenttoolports.ToolPreviewer = (*alerttools.ArrangeHomeLayoutTool)(nil)
