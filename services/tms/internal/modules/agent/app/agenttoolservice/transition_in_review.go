package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*billingtools.TransitionToInReviewTool)(nil)

var _ agentports.ToolValidator = (*billingtools.TransitionToInReviewTool)(nil)

var _ agentports.TargetedTool = (*billingtools.TransitionToInReviewTool)(nil)
