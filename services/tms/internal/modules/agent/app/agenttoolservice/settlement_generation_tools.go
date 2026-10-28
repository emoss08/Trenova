package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*settlementtools.GenerateDriverSettlementTool)(nil)

var _ agenttoolports.ToolValidator = (*settlementtools.GenerateDriverSettlementTool)(nil)

var _ agenttoolports.ToolResultReporter = (*settlementtools.GenerateDriverSettlementTool)(nil)

var _ agenttoolports.ToolPreviewer = (*settlementtools.GenerateDriverBatchTool)(nil)

var _ agenttoolports.ToolValidator = (*settlementtools.GenerateDriverBatchTool)(nil)

var _ agenttoolports.ToolPreviewer = (*settlementtools.GenerateCarrierBatchTool)(nil)

var _ agenttoolports.ToolValidator = (*settlementtools.GenerateCarrierBatchTool)(nil)
