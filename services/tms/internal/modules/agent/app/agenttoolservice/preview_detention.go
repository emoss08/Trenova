package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*masterdatatools.ApproveDetentionTool)(nil)

var _ agenttoolports.ToolPreviewer = (*operationstools.WaiveDetentionTool)(nil)

var _ agenttoolports.ToolPreviewer = (*operationstools.SendDetentionNoticeTool)(nil)
