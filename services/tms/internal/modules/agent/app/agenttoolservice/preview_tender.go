package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/tendertools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*tendertools.TenderToRoutingGuideTool)(nil)

var _ agentports.ToolPreviewer = (*tendertools.TenderToCarriersTool)(nil)
