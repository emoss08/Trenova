package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*shipmenttools.PlaceShipmentHoldTool)(nil)

var _ agentports.ToolPreviewer = (*shipmenttools.ReleaseShipmentHoldTool)(nil)
