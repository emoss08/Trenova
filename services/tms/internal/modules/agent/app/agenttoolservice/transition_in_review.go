package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*billingtools.TransitionToInReviewTool)(nil)

var _ agenttoolports.ToolValidator = (*billingtools.TransitionToInReviewTool)(nil)

var _ agenttoolports.TargetedTool = (*billingtools.TransitionToInReviewTool)(nil)
