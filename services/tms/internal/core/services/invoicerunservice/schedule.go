package invoicerunservice

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/shared/timeutils"
)

// maxCatchUpPeriods bounds how far back a profile that has never billed will
// reach. Without it, a customer switched to statement billing today would
// produce a run for every period since the epoch.
const maxCatchUpPeriods = 24

// Period is a closed billing window: start inclusive, end exclusive.
type Period struct {
	Start int64
	End   int64
}

// PeriodsDue is every period that has closed and not yet been billed.
//
// It returns a slice rather than one period on purpose. A worker outage that
// skips six hourly ticks must still bill every period it missed; returning only
// the latest would silently lose a month's revenue, and the loss would be
// invisible because the next tick would look perfectly healthy.
//
// Boundaries are built in the customer's own zone with time.Date rather than by
// adding 86400, so a DST transition shifts the wall-clock boundary correctly
// instead of moving it by an hour.
func PeriodsDue(profile *customer.CustomerBillingProfile, now int64) []Period {
	if profile == nil || profile.InvoiceDelivery != customer.InvoiceDeliveryConsolidated {
		return nil
	}
	if !profile.BillingCycle.IsPeriodic() {
		return nil
	}

	loc := timeutils.LoadLocation(profile.BillingCycleTimezone)
	nowT := time.Unix(now, 0).In(loc)

	cursor := periodCursor(profile, loc, nowT)
	periods := make([]Period, 0, 2)

	for range maxCatchUpPeriods {
		next := advance(profile, cursor)
		if !next.Before(nowT) {
			break
		}
		periods = append(periods, Period{Start: cursor.Unix(), End: next.Unix()})
		cursor = next
	}

	return periods
}

// CurrentPeriod is the window a statement is accumulating into right now: the
// open period, which has not closed and so has not been billed.
//
// The end is always the customer's own next scheduled boundary, never "now plus
// a cycle". Billing a statement early is allowed, but it must not move the
// cadence — a customer who asked to be billed on the 1st is still billed on the
// 1st after an off-cycle bill on the 14th, and the rest of the month accrues to
// the same period they were expecting.
//
// The start is the later of that boundary and the billing watermark, so freight
// already carried by an off-cycle invoice never shows up as still owing.
func CurrentPeriod(profile *customer.CustomerBillingProfile, now int64) (Period, bool) {
	if profile == nil || profile.InvoiceDelivery != customer.InvoiceDeliveryConsolidated {
		return Period{}, false
	}
	if !profile.BillingCycle.IsPeriodic() {
		return Period{}, false
	}

	loc := timeutils.LoadLocation(profile.BillingCycleTimezone)
	nowT := time.Unix(now, 0).In(loc)

	boundary := periodStartOnOrBefore(profile, loc, nowT)
	period := Period{Start: boundary.Unix(), End: advance(profile, boundary).Unix()}

	if wm := profile.LastBilledPeriodEnd; wm != nil && *wm > period.Start && *wm < period.End {
		period.Start = *wm
	}

	return period, true
}

// periodCursor is where the walk starts: the boundary after the last period
// billed, or the current period's own start for a profile that has never billed.
func periodCursor(
	profile *customer.CustomerBillingProfile,
	loc *time.Location,
	now time.Time,
) time.Time {
	if profile.LastBilledPeriodEnd != nil && *profile.LastBilledPeriodEnd > 0 {
		return time.Unix(*profile.LastBilledPeriodEnd, 0).In(loc)
	}

	return periodStartOnOrBefore(profile, loc, now)
}

// periodStartOnOrBefore is the most recent boundary at or before now.
func periodStartOnOrBefore(
	profile *customer.CustomerBillingProfile,
	loc *time.Location,
	now time.Time,
) time.Time {
	anchor := int(profile.BillingCycleAnchorDay)

	switch profile.BillingCycle {
	case customer.BillingCycleDaily:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	case customer.BillingCycleWeekly, customer.BillingCycleBiWeekly:
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		for int(start.Weekday()) != anchor {
			start = start.AddDate(0, 0, -1)
		}
		return start

	case customer.BillingCycleSemiMonthly:
		day := clampDayOfMonth(anchor)
		if now.Day() >= day {
			return time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, loc)
		}
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)

	case customer.BillingCycleMonthly:
		day := clampDayOfMonth(anchor)
		start := time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, loc)
		if start.After(now) {
			start = start.AddDate(0, -1, 0)
		}
		return start

	case customer.BillingCycleQuarterly:
		day := clampDayOfMonth(anchor)
		month := quarterStartMonth(now.Month())
		start := time.Date(now.Year(), month, day, 0, 0, 0, 0, loc)
		if start.After(now) {
			start = start.AddDate(0, -3, 0)
		}
		return start

	default:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	}
}

// advance is the next boundary after the given one.
func advance(profile *customer.CustomerBillingProfile, from time.Time) time.Time {
	switch profile.BillingCycle {
	case customer.BillingCycleDaily:
		return from.AddDate(0, 0, 1)
	case customer.BillingCycleWeekly:
		return from.AddDate(0, 0, 7)
	case customer.BillingCycleBiWeekly:
		return from.AddDate(0, 0, 14)
	case customer.BillingCycleSemiMonthly:
		// Two boundaries a month: the 1st and the anchor day.
		day := clampDayOfMonth(int(profile.BillingCycleAnchorDay))
		if from.Day() < day {
			return time.Date(from.Year(), from.Month(), day, 0, 0, 0, 0, from.Location())
		}
		next := from.AddDate(0, 1, 0)
		return time.Date(next.Year(), next.Month(), 1, 0, 0, 0, 0, next.Location())
	case customer.BillingCycleMonthly:
		return from.AddDate(0, 1, 0)
	case customer.BillingCycleQuarterly:
		return from.AddDate(0, 3, 0)
	default:
		return from.AddDate(0, 0, 1)
	}
}

// clampDayOfMonth keeps an anchor inside the 1-28 range the domain validates, so
// a boundary never silently shifts in February.
func clampDayOfMonth(day int) int {
	if day < 1 {
		return 1
	}
	if day > 28 {
		return 28
	}
	return day
}

func quarterStartMonth(month time.Month) time.Month {
	return time.Month(((int(month)-1)/3)*3 + 1)
}
