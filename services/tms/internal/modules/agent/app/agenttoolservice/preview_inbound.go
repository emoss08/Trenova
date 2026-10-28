package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/inboundtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*inboundtools.LinkInboundMessageTool)(nil)

var _ agenttoolports.ToolPreviewer = (*inboundtools.MarkInboundMessageTool)(nil)

var _ agenttoolports.ToolPreviewer = (*inboundtools.ReplyToInboundMessageTool)(nil)
