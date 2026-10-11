package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*workforcetools.LinkInboundMessageTool)(nil)

var _ agentports.ToolPreviewer = (*workforcetools.MarkInboundMessageTool)(nil)
