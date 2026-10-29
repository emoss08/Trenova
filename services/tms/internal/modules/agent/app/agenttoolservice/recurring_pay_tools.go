package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
)

var _ agenttoolports.ToolPreviewer = (*settlementtools.RecurringPayTool[recurringpay.RecurringDeduction])(
	nil,
)

var _ agenttoolports.ToolValidator = (*settlementtools.RecurringPayTool[recurringpay.RecurringDeduction])(
	nil,
)

var _ agenttoolports.TargetedTool = (*settlementtools.RecurringPayTool[recurringpay.RecurringDeduction])(
	nil,
)

var _ settlementtools.RecurringBook[recurringpay.RecurringDeduction] = settlementtools.DeductionBook{}

var _ settlementtools.RecurringBook[recurringpay.RecurringEarning] = settlementtools.EarningBook{}
