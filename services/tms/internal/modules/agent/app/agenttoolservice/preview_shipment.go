package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*shipmenttools.AddShipmentCommentTool)(nil)

var _ agentports.ToolPreviewer = (*operationstools.UpdateShipmentTool)(nil)

var _ agentports.ToolPreviewer = (*shipmenttools.CancelShipmentTool)(nil)
