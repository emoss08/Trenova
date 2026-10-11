package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*accountingtools.RetryAccountingSyncTool)(nil)

var _ agentports.ToolPreviewer = (*accountingtools.SkipAccountingSyncTool)(nil)

var _ agentports.ToolPreviewer = (*accountingtools.PauseAccountingSyncTool)(nil)

var _ agentports.ToolPreviewer = (*accountingtools.ResumeAccountingSyncTool)(nil)

var _ agentports.ToolPreviewer = (*accountingtools.RequestAccountingBackfillTool)(nil)
