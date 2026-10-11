package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*billingtools.BillingQueueDecisionTool)(nil)

var _ agentports.ToolValidator = (*billingtools.BillingQueueDecisionTool)(nil)

var _ agentports.TargetedTool = (*billingtools.BillingQueueDecisionTool)(nil)

var _ agentports.ToolPreviewer = (*billingtools.AssignBillerTool)(nil)

var _ agentports.ToolValidator = (*billingtools.AssignBillerTool)(nil)

var _ agentports.TargetedTool = (*billingtools.AssignBillerTool)(nil)
