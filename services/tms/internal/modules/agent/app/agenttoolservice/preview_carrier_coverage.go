package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*carriertools.AssignMoveToCarrierTool)(nil)

var _ agenttoolports.ToolPreviewer = (*carriertools.CancelCarrierAssignmentTool)(nil)

var _ agenttoolports.ToolValidator = (*carriertools.AssignMoveToCarrierTool)(nil)

var _ agenttoolports.ToolValidator = (*carriertools.CancelCarrierAssignmentTool)(nil)
