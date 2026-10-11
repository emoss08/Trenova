package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
)

var _ agentports.ToolPreviewer = (*settlementtools.SettlementBulkTool[driversettlement.Settlement])(
	nil,
)

var _ agentports.ToolValidator = (*settlementtools.SettlementBulkTool[driversettlement.Settlement])(
	nil,
)

var _ agentports.ToolResultReporter = (*settlementtools.SettlementBulkTool[driversettlement.Settlement])(
	nil,
)

var _ agentports.SearchableTool = (*settlementtools.SettlementBulkTool[driversettlement.Settlement])(
	nil,
)
