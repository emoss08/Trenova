package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*settlementtools.OpenEscrowAccountTool)(nil)

var _ agentports.ToolValidator = (*settlementtools.OpenEscrowAccountTool)(nil)

var _ agentports.ToolPreviewer = (*settlementtools.UpdateEscrowAccountTool)(nil)

var _ agentports.ToolValidator = (*settlementtools.UpdateEscrowAccountTool)(nil)

var _ agentports.TargetedTool = (*settlementtools.UpdateEscrowAccountTool)(nil)

var _ agentports.ToolPreviewer = (*settlementtools.AdjustEscrowAccountTool)(nil)

var _ agentports.ToolValidator = (*settlementtools.AdjustEscrowAccountTool)(nil)

var _ agentports.TargetedTool = (*settlementtools.AdjustEscrowAccountTool)(nil)

var _ agentports.ToolPreviewer = (*settlementtools.CloseEscrowAccountTool)(nil)

var _ agentports.ToolValidator = (*settlementtools.CloseEscrowAccountTool)(nil)

var _ agentports.TargetedTool = (*settlementtools.CloseEscrowAccountTool)(nil)
