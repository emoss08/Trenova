package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/inboundtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*inboundtools.LinkInboundMessageTool)(nil)

var _ agentports.ToolPreviewer = (*inboundtools.MarkInboundMessageTool)(nil)

var _ agentports.ToolPreviewer = (*inboundtools.ReplyToInboundMessageTool)(nil)
