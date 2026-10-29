package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/findingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*findingtools.ApproveDetentionTool)(nil)

var _ agenttoolports.ToolPreviewer = (*operationstools.WaiveDetentionTool)(nil)

var _ agenttoolports.ToolPreviewer = (*operationstools.SendDetentionNoticeTool)(nil)
