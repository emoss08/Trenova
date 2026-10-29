package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/intaketools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentcommenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*shipmentcommenttools.AddShipmentCommentTool)(nil)

var _ agenttoolports.ToolPreviewer = (*intaketools.UpdateShipmentTool)(nil)

var _ agenttoolports.ToolPreviewer = (*shipmenttools.CancelShipmentTool)(nil)
