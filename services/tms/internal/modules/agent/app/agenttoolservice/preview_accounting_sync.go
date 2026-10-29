package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingsynctools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*accountingsynctools.RetryAccountingSyncTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingsynctools.SkipAccountingSyncTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingsynctools.PauseAccountingSyncTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingsynctools.ResumeAccountingSyncTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingsynctools.RequestAccountingBackfillTool)(nil)
