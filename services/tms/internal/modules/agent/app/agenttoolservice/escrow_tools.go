package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*settlementtools.OpenEscrowAccountTool)(nil)

var _ agenttoolports.ToolValidator = (*settlementtools.OpenEscrowAccountTool)(nil)

var _ agenttoolports.ToolPreviewer = (*settlementtools.UpdateEscrowAccountTool)(nil)

var _ agenttoolports.ToolValidator = (*settlementtools.UpdateEscrowAccountTool)(nil)

var _ agenttoolports.TargetedTool = (*settlementtools.UpdateEscrowAccountTool)(nil)

var _ agenttoolports.ToolPreviewer = (*settlementtools.AdjustEscrowAccountTool)(nil)

var _ agenttoolports.ToolValidator = (*settlementtools.AdjustEscrowAccountTool)(nil)

var _ agenttoolports.TargetedTool = (*settlementtools.AdjustEscrowAccountTool)(nil)

var _ agenttoolports.ToolPreviewer = (*settlementtools.CloseEscrowAccountTool)(nil)

var _ agenttoolports.ToolValidator = (*settlementtools.CloseEscrowAccountTool)(nil)

var _ agenttoolports.TargetedTool = (*settlementtools.CloseEscrowAccountTool)(nil)
