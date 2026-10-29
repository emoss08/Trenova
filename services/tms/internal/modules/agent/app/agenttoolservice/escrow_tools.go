package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytermtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*paytermtools.OpenEscrowAccountTool)(nil)

var _ agenttoolports.ToolValidator = (*paytermtools.OpenEscrowAccountTool)(nil)

var _ agenttoolports.ToolPreviewer = (*paytermtools.UpdateEscrowAccountTool)(nil)

var _ agenttoolports.ToolValidator = (*paytermtools.UpdateEscrowAccountTool)(nil)

var _ agenttoolports.TargetedTool = (*paytermtools.UpdateEscrowAccountTool)(nil)

var _ agenttoolports.ToolPreviewer = (*paytermtools.AdjustEscrowAccountTool)(nil)

var _ agenttoolports.ToolValidator = (*paytermtools.AdjustEscrowAccountTool)(nil)

var _ agenttoolports.TargetedTool = (*paytermtools.AdjustEscrowAccountTool)(nil)

var _ agenttoolports.ToolPreviewer = (*paytermtools.CloseEscrowAccountTool)(nil)

var _ agenttoolports.ToolValidator = (*paytermtools.CloseEscrowAccountTool)(nil)

var _ agenttoolports.TargetedTool = (*paytermtools.CloseEscrowAccountTool)(nil)
