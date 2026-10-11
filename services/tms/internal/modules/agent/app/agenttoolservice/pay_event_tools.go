package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*paytools.PayEventHoldTool)(nil)

var _ agentports.ToolValidator = (*paytools.PayEventHoldTool)(nil)

var _ agentports.ToolPreviewer = (*paytools.PayEventTransferTool)(nil)

var _ agentports.ToolValidator = (*paytools.PayEventTransferTool)(nil)

var _ agentports.TargetedTool = (*paytools.PayEventTransferTool)(nil)
