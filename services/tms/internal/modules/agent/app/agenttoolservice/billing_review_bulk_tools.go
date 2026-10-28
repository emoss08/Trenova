package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*billingtools.AssignBillersTool)(nil)

var _ agenttoolports.ToolValidator = (*billingtools.AssignBillersTool)(nil)

var _ agenttoolports.ToolResultReporter = (*billingtools.AssignBillersTool)(nil)

var _ agenttoolports.ToolPreviewer = (*billingtools.TransitionItemsToInReviewTool)(nil)

var _ agenttoolports.ToolValidator = (*billingtools.TransitionItemsToInReviewTool)(nil)

var _ agenttoolports.ToolResultReporter = (*billingtools.TransitionItemsToInReviewTool)(nil)
