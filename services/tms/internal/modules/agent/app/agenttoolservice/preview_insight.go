package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/insighttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*insighttools.DismissInsightTool)(nil)
