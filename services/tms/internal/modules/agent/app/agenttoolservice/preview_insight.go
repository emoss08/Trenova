package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/insighttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*insighttools.DismissInsightTool)(nil)
