package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*accountingtools.ResolveAccountingDriftTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingtools.DismissAccountingDriftTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingtools.CheckAccountingDriftTool)(nil)
