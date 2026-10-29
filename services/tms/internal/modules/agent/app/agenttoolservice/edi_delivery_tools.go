package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*editools.RetryEDIMessageDeliveryTool)(nil)

var _ agenttoolports.ToolValidator = (*editools.RetryEDIMessageDeliveryTool)(nil)

var _ agenttoolports.ToolResultReporter = (*editools.RetryEDIMessageDeliveryTool)(nil)

var _ agenttoolports.ToolPreviewer = (*editools.ReplayEDIMessageTool)(nil)

var _ agenttoolports.ToolValidator = (*editools.ReplayEDIMessageTool)(nil)

var _ agenttoolports.TargetedTool = (*editools.ReplayEDIMessageTool)(nil)

var _ agenttoolports.ToolPreviewer = (*editools.ReprocessEDIInboundFilesTool)(nil)

var _ agenttoolports.ToolValidator = (*editools.ReprocessEDIInboundFilesTool)(nil)

var _ agenttoolports.ToolResultReporter = (*editools.ReprocessEDIInboundFilesTool)(nil)
