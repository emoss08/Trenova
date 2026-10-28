package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
)

var _ agenttoolports.ToolPreviewer = (*settlementtools.SettlementBulkTool[driversettlement.Settlement])(
	nil,
)

var _ agenttoolports.ToolValidator = (*settlementtools.SettlementBulkTool[driversettlement.Settlement])(
	nil,
)

var _ agenttoolports.ToolResultReporter = (*settlementtools.SettlementBulkTool[driversettlement.Settlement])(
	nil,
)

var _ agenttoolports.SearchableTool = (*settlementtools.SettlementBulkTool[driversettlement.Settlement])(
	nil,
)
