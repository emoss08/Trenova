package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementgenerationtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

var _ agenttoolports.ToolPreviewer = (*settlementgenerationtools.GenerateDriverSettlementTool)(nil)

var _ agenttoolports.ToolValidator = (*settlementgenerationtools.GenerateDriverSettlementTool)(nil)

var _ agenttoolports.ToolResultReporter = (*settlementgenerationtools.GenerateDriverSettlementTool)(
	nil,
)

var _ agenttoolports.ToolPreviewer = (*settlementgenerationtools.GenerateDriverBatchTool)(nil)

var _ agenttoolports.ToolValidator = (*settlementgenerationtools.GenerateDriverBatchTool)(nil)

var _ agenttoolports.ToolPreviewer = (*settlementgenerationtools.GenerateCarrierBatchTool)(nil)

var _ agenttoolports.ToolValidator = (*settlementgenerationtools.GenerateCarrierBatchTool)(nil)
