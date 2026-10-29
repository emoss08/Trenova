package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*shipmenttools.PlaceShipmentHoldTool)(nil)

var _ agenttoolports.ToolPreviewer = (*shipmenttools.ReleaseShipmentHoldTool)(nil)
