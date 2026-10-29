package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/paytermtools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
)

var _ agenttoolports.ToolPreviewer = (*paytermtools.RecurringPayTool[recurringpay.RecurringDeduction])(
	nil,
)

var _ agenttoolports.ToolValidator = (*paytermtools.RecurringPayTool[recurringpay.RecurringDeduction])(
	nil,
)

var _ agenttoolports.TargetedTool = (*paytermtools.RecurringPayTool[recurringpay.RecurringDeduction])(
	nil,
)

var _ paytermtools.RecurringBook[recurringpay.RecurringDeduction] = paytermtools.DeductionBook{}

var _ paytermtools.RecurringBook[recurringpay.RecurringEarning] = paytermtools.EarningBook{}
