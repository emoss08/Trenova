package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*operationstools.EmailCustomerTool)(nil)

var _ agentports.ToolPreviewer = (*workforcetools.RequestMissingDocsTool)(nil)

var _ agentports.ToolPreviewer = (*workforcetools.ReplyToInboundMessageTool)(nil)
