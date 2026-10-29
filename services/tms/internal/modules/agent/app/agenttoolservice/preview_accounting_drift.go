package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingdrifttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*accountingdrifttools.ResolveAccountingDriftTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingdrifttools.DismissAccountingDriftTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingdrifttools.CheckAccountingDriftTool)(nil)
