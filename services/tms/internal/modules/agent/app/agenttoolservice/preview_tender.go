package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/tendertools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*tendertools.TenderToRoutingGuideTool)(nil)

var _ agenttoolports.ToolPreviewer = (*tendertools.TenderToCarriersTool)(nil)
