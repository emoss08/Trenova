package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*billingtools.BillingQueueDecisionTool)(nil)

var _ agenttoolports.ToolValidator = (*billingtools.BillingQueueDecisionTool)(nil)

var _ agenttoolports.TargetedTool = (*billingtools.BillingQueueDecisionTool)(nil)

var _ agenttoolports.ToolPreviewer = (*billingtools.AssignBillerTool)(nil)

var _ agenttoolports.ToolValidator = (*billingtools.AssignBillerTool)(nil)

var _ agenttoolports.TargetedTool = (*billingtools.AssignBillerTool)(nil)
