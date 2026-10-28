package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*masterdatatools.RememberTool)(nil)

var _ agenttoolports.ToolPreviewer = (*masterdatatools.ForgetMemoryTool)(nil)
