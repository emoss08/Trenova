package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*paytools.PayEventHoldTool)(nil)

var _ agenttoolports.ToolValidator = (*paytools.PayEventHoldTool)(nil)

var _ agenttoolports.ToolPreviewer = (*paytools.PayEventTransferTool)(nil)

var _ agenttoolports.ToolValidator = (*paytools.PayEventTransferTool)(nil)

var _ agenttoolports.TargetedTool = (*paytools.PayEventTransferTool)(nil)
