package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
)

var _ settlementtools.SettlementBook[driversettlement.Settlement] = (*driversettlementservice.Service)(
	nil,
)

var _ settlementtools.SettlementBook[carriersettlement.CarrierSettlement] = (*carriersettlementservice.Service)(
	nil,
)

var _ agenttoolports.ToolPreviewer = (*settlementtools.SettlementDecisionTool[driversettlement.Settlement])(
	nil,
)

var _ agenttoolports.ToolValidator = (*settlementtools.SettlementDecisionTool[driversettlement.Settlement])(
	nil,
)

var _ agenttoolports.TargetedTool = (*settlementtools.SettlementDecisionTool[driversettlement.Settlement])(
	nil,
)
