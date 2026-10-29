package agenttooltrust

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/shared/intutils"
)

const (
	HeldByMoneyUnbounded    = "money_unbounded"
	HeldByMoneyUnproven     = "money_unproven"
	HeldByMoneyExactMatch   = "money_exact_match"
	HeldByMoneyRecordState  = "money_record_state"
	HeldByMoneyAgentLine    = "money_agent_line"
	HeldByMoneyLimitsUnread = "money_limits_unread"
	HeldByMoneyCurrency     = "money_currency"
	HeldByMoneyCallCeiling  = "money_call_ceiling"
	HeldByMoneyRunCalls     = "money_run_calls"
	HeldByMoneyRunAmount    = "money_run_amount"
	HeldByMoneyDayCalls     = "money_day_calls"
	HeldByMoneyDayAmount    = "money_day_amount"
)

func MoneyHeldByKeys() []string {
	return []string{
		HeldByMoneyUnbounded,
		HeldByMoneyUnproven,
		HeldByMoneyExactMatch,
		HeldByMoneyRecordState,
		HeldByMoneyAgentLine,
		HeldByMoneyLimitsUnread,
		HeldByMoneyCurrency,
		HeldByMoneyCallCeiling,
		HeldByMoneyRunCalls,
		HeldByMoneyRunAmount,
		HeldByMoneyDayCalls,
		HeldByMoneyDayAmount,
	}
}

func IsMoneyHeldByKey(key string) bool {
	return slices.Contains(MoneyHeldByKeys(), key)
}

func MoneyConditionHeldByKeys() []string {
	return []string{
		HeldByMoneyUnproven,
		HeldByMoneyExactMatch,
		HeldByMoneyRecordState,
		HeldByMoneyAgentLine,
	}
}

type MoneyLimits struct {
	CallCeilingMinor int64
	RunCalls         int
	RunAmountMinor   int64
	DayCalls         int
	DayAmountMinor   int64
}

type MoneyUse struct {
	Calls       int
	AmountMinor int64
}

func (u MoneyUse) Plus(other MoneyUse) MoneyUse {
	return MoneyUse{Calls: u.Calls + other.Calls, AmountMinor: u.AmountMinor + other.AmountMinor}
}

type MoneyAllowance struct {
	Limits   MoneyLimits
	Currency string
	Run      MoneyUse
	Day      MoneyUse
}

func (a *MoneyAllowance) Exceeded(amountMinor int64, currency string) string {
	amount := intutils.Abs(amountMinor)
	switch {
	case !a.Counts(currency):
		return HeldByMoneyCurrency
	case amount < 0 || a.Limits.CallCeilingMinor <= 0 || amount > a.Limits.CallCeilingMinor:
		return HeldByMoneyCallCeiling
	case a.Run.Calls >= a.Limits.RunCalls:
		return HeldByMoneyRunCalls
	case exceeds(a.Run.AmountMinor, amount, a.Limits.RunAmountMinor):
		return HeldByMoneyRunAmount
	case a.Day.Calls >= a.Limits.DayCalls:
		return HeldByMoneyDayCalls
	case exceeds(a.Day.AmountMinor, amount, a.Limits.DayAmountMinor):
		return HeldByMoneyDayAmount
	default:
		return ""
	}
}

func exceeds(used, amount, limit int64) bool {
	return used < 0 || used > limit || amount > limit-used
}

func (a *MoneyAllowance) Counts(currency string) bool {
	functional := strings.TrimSpace(a.Currency)

	return functional != "" && strings.EqualFold(strings.TrimSpace(currency), functional)
}
