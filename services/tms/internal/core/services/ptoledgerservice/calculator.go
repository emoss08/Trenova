package ptoledgerservice

import (
	"fmt"
	"sort"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/shopspring/decimal"
)

const (
	periodKeyMonthly     = "M:"
	periodKeyAnnualGrant = "A:"
	periodKeyCarryover   = "C:"
	periodKeyExpiry      = "X:"
	periodKeyPayPeriod   = "P:"
	periodKeyTermination = "T:"

	DefaultLookbackMonths = 24

	orderExpiryCarryover = 0
	orderExpiryCarried   = 1
	orderAccrual         = 2

	maxPayPeriodsPerRun = 400
)

// PayPeriodSpec is the slice of the settlement control the per-pay-period
// method needs: how long a period is and which weekday closes it.
type PayPeriodSpec struct {
	Frequency    tenant.PayPeriodFrequency
	EndDayOfWeek int
}

type CalcInput struct {
	Rule            worker.PTOPolicyRule
	YearBasis       worker.PTOYearBasis
	WaitingDays     int32
	HireDate        int64
	TerminationDate *int64
	Loc             *time.Location
	AsOf            int64
	LastAccrualKey  string
	LastRolloverKey string
	LookbackMonths  int
	PayPeriod       *PayPeriodSpec
}

type PlannedEntry struct {
	EntryType   worker.PTOLedgerEntryType
	PeriodKey   string
	EffectiveAt int64
	NominalDays decimal.Decimal
	// MaxBalanceDays is the cap in force at EffectiveAt once tenure tiers are
	// applied; nil-valid means the rule has no cap at that tenure.
	MaxBalanceDays decimal.NullDecimal
	Deferred       bool
	Order          int
}

func (e PlannedEntry) IsRollover() bool {
	return e.EntryType == worker.PTOLedgerEntryExpiry
}

func dateOf(unix int64, loc *time.Location) time.Time {
	t := time.Unix(unix, 0).In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func anniversary(hire time.Time, year int, loc *time.Location) time.Time {
	day := hire.Day()
	if hire.Month() == time.February && day == 29 {
		day = 28
	}
	return time.Date(year, hire.Month(), day, 0, 0, 0, 0, loc)
}

func firstOfMonthOnOrAfter(t time.Time, loc *time.Location) time.Time {
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
	if first.Before(t) {
		return first.AddDate(0, 1, 0)
	}
	return first
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func yearStarts(in CalcInput, hire, floor, horizon time.Time, loc *time.Location) []time.Time {
	starts := make([]time.Time, 0, 4)
	for year := floor.Year(); year <= horizon.Year()+1; year++ {
		var start time.Time
		if in.YearBasis == worker.PTOYearBasisHireAnniversary {
			start = anniversary(hire, year, loc)
		} else {
			start = time.Date(year, time.January, 1, 0, 0, 0, 0, loc)
		}
		if start.After(floor) && !start.After(horizon) {
			starts = append(starts, start)
		}
	}
	return starts
}

// accrualAt resolves the tiered amount and cap in force on a given day.
func accrualAt(in CalcInput, at time.Time, loc *time.Location) (decimal.Decimal, decimal.NullDecimal) {
	tenure := worker.TenureMonths(in.HireDate, at.Unix(), loc)
	return in.Rule.AmountFor(tenure), in.Rule.MaxBalanceFor(tenure)
}

// payPeriodEnds lists the pay-period closing dates (exclusive end, midnight)
// strictly after `from` and not after `horizon`, walking the settlement
// calendar forward from the period that contains `from`.
func payPeriodEnds(spec *PayPeriodSpec, from, horizon time.Time) []time.Time {
	if spec == nil || !spec.Frequency.IsValid() {
		return nil
	}
	ends := make([]time.Time, 0, 32)
	bounds := settlementshared.ResolvePeriod(spec.Frequency, spec.EndDayOfWeek, 0, from.Unix())
	end := time.Unix(bounds.PeriodEnd, 0).UTC()
	for i := 0; i < maxPayPeriodsPerRun && !end.After(horizon); i++ {
		if end.After(from) {
			ends = append(ends, end)
		}
		switch spec.Frequency {
		case tenant.PayPeriodFrequencyWeekly:
			end = end.AddDate(0, 0, 7)
		case tenant.PayPeriodFrequencyBiweekly:
			end = end.AddDate(0, 0, 14)
		case tenant.PayPeriodFrequencyMonthly:
			end = end.AddDate(0, 1, 0)
		default:
			end = end.AddDate(0, 0, 7)
		}
	}
	return ends
}

func Schedule(in CalcInput) []PlannedEntry {
	loc := in.Loc
	if loc == nil {
		loc = time.UTC
	}
	lookback := in.LookbackMonths
	if lookback <= 0 {
		lookback = DefaultLookbackMonths
	}

	hire := dateOf(in.HireDate, loc)
	eligibleFrom := hire.AddDate(0, 0, int(in.WaitingDays))
	asOf := dateOf(in.AsOf, loc)
	horizon := asOf
	if in.TerminationDate != nil {
		horizon = minTime(horizon, dateOf(*in.TerminationDate, loc))
	}
	floor := maxTime(eligibleFrom.AddDate(0, 0, -1), asOf.AddDate(0, -lookback, 0))
	if horizon.Before(eligibleFrom) {
		return nil
	}

	plan := make([]PlannedEntry, 0, 8)

	for _, start := range yearStarts(in, hire, floor, horizon, loc) {
		label := fmt.Sprintf("%d", start.Year())
		carryKey := periodKeyCarryover + label
		if carryKey > in.LastRolloverKey {
			if in.Rule.CarryoverCapDays.Valid {
				plan = append(plan, PlannedEntry{
					EntryType:   worker.PTOLedgerEntryExpiry,
					PeriodKey:   carryKey,
					EffectiveAt: start.Unix(),
					Deferred:    true,
					Order:       orderExpiryCarryover,
				})
				if in.Rule.CarryoverExpiryDays > 0 {
					expiresAt := start.AddDate(0, 0, int(in.Rule.CarryoverExpiryDays))
					if !expiresAt.After(horizon) {
						plan = append(plan, PlannedEntry{
							EntryType:   worker.PTOLedgerEntryExpiry,
							PeriodKey:   periodKeyExpiry + label,
							EffectiveAt: expiresAt.Unix(),
							Deferred:    true,
							Order:       orderExpiryCarried,
						})
					}
				}
			}
		}

		if in.Rule.AccrualMethod == worker.PTOAccrualMethodFixedAnnualGrant &&
			in.Rule.Accrues() {
			grantKey := periodKeyAnnualGrant + label
			if grantKey > in.LastAccrualKey && !start.Before(eligibleFrom) {
				amount, cap := accrualAt(in, start, loc)
				plan = append(plan, PlannedEntry{
					EntryType:      worker.PTOLedgerEntryAccrual,
					PeriodKey:      grantKey,
					EffectiveAt:    start.Unix(),
					NominalDays:    amount,
					MaxBalanceDays: cap,
					Order:          orderAccrual,
				})
			}
		}
	}

	if in.Rule.AccrualMethod == worker.PTOAccrualMethodMonthly && in.Rule.Accrues() {
		for ms := firstOfMonthOnOrAfter(maxTime(eligibleFrom, floor.AddDate(0, 0, 1)), loc); !ms.After(horizon); ms = ms.AddDate(0, 1, 0) {
			key := periodKeyMonthly + ms.Format("2006-01")
			if key <= in.LastAccrualKey {
				continue
			}
			amount, cap := accrualAt(in, ms, loc)
			plan = append(plan, PlannedEntry{
				EntryType:      worker.PTOLedgerEntryAccrual,
				PeriodKey:      key,
				EffectiveAt:    ms.Unix(),
				NominalDays:    amount,
				MaxBalanceDays: cap,
				Order:          orderAccrual,
			})
		}
	}

	if in.Rule.AccrualMethod == worker.PTOAccrualMethodPerPayPeriod && in.Rule.Accrues() {
		from := maxTime(eligibleFrom.AddDate(0, 0, -1), floor)
		for _, end := range payPeriodEnds(in.PayPeriod, from, horizon) {
			key := periodKeyPayPeriod + end.Format("2006-01-02")
			if key <= in.LastAccrualKey {
				continue
			}
			amount, cap := accrualAt(in, end, loc)
			plan = append(plan, PlannedEntry{
				EntryType:      worker.PTOLedgerEntryAccrual,
				PeriodKey:      key,
				EffectiveAt:    end.Unix(),
				NominalDays:    amount,
				MaxBalanceDays: cap,
				Order:          orderAccrual,
			})
		}
	}

	sort.SliceStable(plan, func(i, j int) bool {
		if plan[i].EffectiveAt != plan[j].EffectiveAt {
			return plan[i].EffectiveAt < plan[j].EffectiveAt
		}
		return plan[i].Order < plan[j].Order
	})

	return plan
}

func ProjectedAccrual(
	plan []PlannedEntry,
	balance, carried decimal.Decimal,
	rule worker.PTOPolicyRule,
) decimal.Decimal {
	running := balance
	for _, entry := range plan {
		switch entry.EntryType {
		case worker.PTOLedgerEntryAccrual:
			amount := entry.NominalDays
			cap := entry.MaxBalanceDays
			if !cap.Valid {
				cap = rule.MaxBalanceDays
			}
			if cap.Valid {
				room := cap.Decimal.Sub(running)
				if room.LessThan(amount) {
					amount = room
				}
			}
			if amount.IsPositive() {
				running = running.Add(amount)
			}
		case worker.PTOLedgerEntryExpiry:
			if entry.PeriodKey[:2] == periodKeyCarryover && rule.CarryoverCapDays.Valid {
				excess := running.Sub(rule.CarryoverCapDays.Decimal)
				if excess.IsPositive() {
					running = running.Sub(excess)
					carried = rule.CarryoverCapDays.Decimal
				} else {
					carried = running
				}
			} else if entry.PeriodKey[:2] == periodKeyExpiry {
				expire := decimal.Min(carried, running)
				if expire.IsPositive() {
					running = running.Sub(expire)
				}
				carried = decimal.Zero
			}
		case worker.PTOLedgerEntryOpeningBalance, worker.PTOLedgerEntryUsage,
			worker.PTOLedgerEntryReversal, worker.PTOLedgerEntryAdjustment,
			worker.PTOLedgerEntryCarryover, worker.PTOLedgerEntryPayout,
			worker.PTOLedgerEntryForfeiture:
		}
	}
	return running.Sub(balance)
}
