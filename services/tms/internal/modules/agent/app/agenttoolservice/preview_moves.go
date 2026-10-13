package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/movetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var (
	_ agentports.ToolPreviewer = (*movetools.RecordStopActualTool)(nil)
	_ agentports.ToolPreviewer = (*movetools.LogCheckCallTool)(nil)
)
