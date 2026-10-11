package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*billingtools.ApproveBillingQueueItemTool)(nil)

var _ agentports.ToolResultReporter = (*billingtools.ApproveBillingQueueItemTool)(nil)
