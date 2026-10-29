package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*accountingtools.CheckAccountingConnectionTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingtools.SetAccountingMappingTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingtools.ClearAccountingMappingTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingtools.CreateAccountingReferenceRecordTool)(nil)

var _ agenttoolports.ToolPreviewer = (*accountingtools.RefreshAccountingReferenceDataTool)(nil)
