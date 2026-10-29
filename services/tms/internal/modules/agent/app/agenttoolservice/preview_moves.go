package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/movetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var (
	_ agenttoolports.ToolPreviewer = (*movetools.RecordStopActualTool)(nil)
	_ agenttoolports.ToolPreviewer = (*movetools.LogCheckCallTool)(nil)
)
