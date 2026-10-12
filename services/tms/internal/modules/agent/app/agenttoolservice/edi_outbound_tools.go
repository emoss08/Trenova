package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*editools.SendEDILoadTenderTool)(nil)

var _ agentports.ToolValidator = (*editools.SendEDILoadTenderTool)(nil)

var _ agentports.TargetedTool = (*editools.SendEDILoadTenderTool)(nil)

var _ agentports.ToolPreviewer = (*editools.SendEDIStatusUpdateTool)(nil)

var _ agentports.ToolValidator = (*editools.SendEDIStatusUpdateTool)(nil)
