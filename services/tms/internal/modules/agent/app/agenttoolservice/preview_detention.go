package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*masterdatatools.ApproveDetentionTool)(nil)

var _ agentports.ToolPreviewer = (*operationstools.WaiveDetentionTool)(nil)

var _ agentports.ToolPreviewer = (*operationstools.SendDetentionNoticeTool)(nil)
