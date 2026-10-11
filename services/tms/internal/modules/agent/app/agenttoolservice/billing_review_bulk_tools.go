package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*billingtools.AssignBillersTool)(nil)

var _ agentports.ToolValidator = (*billingtools.AssignBillersTool)(nil)

var _ agentports.ToolResultReporter = (*billingtools.AssignBillersTool)(nil)

var _ agentports.ToolPreviewer = (*billingtools.TransitionItemsToInReviewTool)(nil)

var _ agentports.ToolValidator = (*billingtools.TransitionItemsToInReviewTool)(nil)

var _ agentports.ToolResultReporter = (*billingtools.TransitionItemsToInReviewTool)(nil)
