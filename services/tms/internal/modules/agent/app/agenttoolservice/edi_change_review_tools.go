package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*editools.EdiChangeReviewTool)(nil)

var _ agenttoolports.ToolValidator = (*editools.EdiChangeReviewTool)(nil)

var _ agenttoolports.TargetedTool = (*editools.EdiChangeReviewTool)(nil)
