package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

var _ agentports.ToolPreviewer = (*settlementtools.GenerateDriverSettlementTool)(nil)

var _ agentports.ToolValidator = (*settlementtools.GenerateDriverSettlementTool)(nil)

var _ agentports.ToolResultReporter = (*settlementtools.GenerateDriverSettlementTool)(nil)

var _ agentports.ToolPreviewer = (*settlementtools.GenerateDriverBatchTool)(nil)

var _ agentports.ToolValidator = (*settlementtools.GenerateDriverBatchTool)(nil)

var _ agentports.ToolPreviewer = (*settlementtools.GenerateCarrierBatchTool)(nil)

var _ agentports.ToolValidator = (*settlementtools.GenerateCarrierBatchTool)(nil)
