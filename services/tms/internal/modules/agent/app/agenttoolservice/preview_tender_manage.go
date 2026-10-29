package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/tendertools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*tendertools.CancelTenderTool)(nil)

var _ agenttoolports.ToolPreviewer = (*tendertools.RecordTenderResponseTool)(nil)

var _ agenttoolports.ToolValidator = (*tendertools.CancelTenderTool)(nil)

var _ agenttoolports.ToolValidator = (*tendertools.RecordTenderResponseTool)(nil)
