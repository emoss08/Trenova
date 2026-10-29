package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountinginboundtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*accountinginboundtools.ApplyAccountingInboundChangeTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountinginboundtools.IgnoreAccountingInboundChangeTool)(
	nil,
)
