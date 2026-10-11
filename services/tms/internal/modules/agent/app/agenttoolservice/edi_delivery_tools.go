package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*editools.RetryEDIMessageDeliveryTool)(nil)

var _ agentports.ToolValidator = (*editools.RetryEDIMessageDeliveryTool)(nil)

var _ agentports.ToolResultReporter = (*editools.RetryEDIMessageDeliveryTool)(nil)

var _ agentports.ToolPreviewer = (*editools.ReplayEDIMessageTool)(nil)

var _ agentports.ToolValidator = (*editools.ReplayEDIMessageTool)(nil)

var _ agentports.TargetedTool = (*editools.ReplayEDIMessageTool)(nil)

var _ agentports.ToolPreviewer = (*editools.ReprocessEDIInboundFilesTool)(nil)

var _ agentports.ToolValidator = (*editools.ReprocessEDIInboundFilesTool)(nil)

var _ agentports.ToolResultReporter = (*editools.ReprocessEDIInboundFilesTool)(nil)
