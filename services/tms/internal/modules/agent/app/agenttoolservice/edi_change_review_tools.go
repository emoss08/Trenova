package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/editools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*editools.EdiChangeReviewTool)(nil)

var _ agentports.ToolValidator = (*editools.EdiChangeReviewTool)(nil)

var _ agentports.TargetedTool = (*editools.EdiChangeReviewTool)(nil)
