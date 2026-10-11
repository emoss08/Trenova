package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/tendertools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*tendertools.CancelTenderTool)(nil)

var _ agentports.ToolPreviewer = (*tendertools.RecordTenderResponseTool)(nil)

var _ agentports.ToolValidator = (*tendertools.CancelTenderTool)(nil)

var _ agentports.ToolValidator = (*tendertools.RecordTenderResponseTool)(nil)
