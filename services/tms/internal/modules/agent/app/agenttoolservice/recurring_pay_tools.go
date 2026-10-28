package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
)

var _ agenttoolports.ToolPreviewer = (*settlementtools.RecurringPayTool[driverpay.RecurringDeduction])(
	nil,
)

var _ agenttoolports.ToolValidator = (*settlementtools.RecurringPayTool[driverpay.RecurringDeduction])(
	nil,
)

var _ agenttoolports.TargetedTool = (*settlementtools.RecurringPayTool[driverpay.RecurringDeduction])(
	nil,
)

var _ settlementtools.RecurringBook[driverpay.RecurringDeduction] = settlementtools.DeductionBook{}

var _ settlementtools.RecurringBook[driverpay.RecurringEarning] = settlementtools.EarningBook{}
