package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*shipmenttools.AddShipmentCommentTool)(nil)

var _ agenttoolports.ToolPreviewer = (*operationstools.UpdateShipmentTool)(nil)

var _ agenttoolports.ToolPreviewer = (*shipmenttools.CancelShipmentTool)(nil)
