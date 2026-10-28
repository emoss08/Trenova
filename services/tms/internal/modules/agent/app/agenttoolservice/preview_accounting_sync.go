package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*accountingtools.RetryAccountingSyncTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingtools.SkipAccountingSyncTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingtools.PauseAccountingSyncTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingtools.ResumeAccountingSyncTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingtools.RequestAccountingBackfillTool)(nil)
