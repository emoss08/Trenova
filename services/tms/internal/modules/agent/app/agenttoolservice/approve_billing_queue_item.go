package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*billingtools.ApproveBillingQueueItemTool)(nil)

var _ agenttoolports.ToolResultReporter = (*billingtools.ApproveBillingQueueItemTool)(nil)
