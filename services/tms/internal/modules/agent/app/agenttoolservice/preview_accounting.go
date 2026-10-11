package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/accountingtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*accountingtools.CheckAccountingConnectionTool)(nil)

var _ agentports.ToolPreviewer = (*accountingtools.SetAccountingMappingTool)(nil)

var _ agentports.ToolPreviewer = (*accountingtools.ClearAccountingMappingTool)(nil)

var _ agentports.ToolPreviewer = (*accountingtools.CreateAccountingReferenceRecordTool)(nil)

var _ agentports.ToolPreviewer = (*accountingtools.RefreshAccountingReferenceDataTool)(nil)
