package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*editools.SendEDILoadTenderTool)(nil)

var _ agenttoolports.ToolValidator = (*editools.SendEDILoadTenderTool)(nil)

var _ agenttoolports.TargetedTool = (*editools.SendEDILoadTenderTool)(nil)

var _ agenttoolports.ToolPreviewer = (*editools.SendEDIStatusUpdateTool)(nil)

var _ agenttoolports.ToolValidator = (*editools.SendEDIStatusUpdateTool)(nil)
