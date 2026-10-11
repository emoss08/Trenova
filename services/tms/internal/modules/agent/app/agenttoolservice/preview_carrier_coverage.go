package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*carriertools.AssignMoveToCarrierTool)(nil)

var _ agentports.ToolPreviewer = (*carriertools.CancelCarrierAssignmentTool)(nil)

var _ agentports.ToolValidator = (*carriertools.AssignMoveToCarrierTool)(nil)

var _ agentports.ToolValidator = (*carriertools.CancelCarrierAssignmentTool)(nil)
