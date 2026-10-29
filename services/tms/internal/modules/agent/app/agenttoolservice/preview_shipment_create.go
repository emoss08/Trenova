package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/intaketools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*intaketools.CreateShipmentTool)(nil)
