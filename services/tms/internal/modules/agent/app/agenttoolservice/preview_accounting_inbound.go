package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*accountingtools.ApplyAccountingInboundChangeTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingtools.IgnoreAccountingInboundChangeTool)(nil)
