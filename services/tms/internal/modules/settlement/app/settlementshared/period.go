package settlementshared

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/settlementcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

type PeriodSchedule struct {
	Frequency    settlementcontrol.PayPeriodFrequency
	EndDayOfWeek int
	PayDelayDays int
	Location     *time.Location
}

func ResolveRequestedPeriod(
	schedule PeriodSchedule,
	periodStart, periodEnd, now int64,
) (PeriodBounds, error) {
	bounds := PeriodBounds{PeriodStart: periodStart, PeriodEnd: periodEnd}
	if bounds.PeriodStart == 0 || bounds.PeriodEnd == 0 {
		bounds = ResolvePeriod(
			schedule.Frequency,
			schedule.EndDayOfWeek,
			schedule.PayDelayDays,
			now,
			schedule.Location,
		)
	} else {
		bounds.PayDate = time.Unix(bounds.PeriodEnd, 0).In(locationOrUTC(schedule.Location)).
			AddDate(0, 0, schedule.PayDelayDays).Unix()
	}
	if bounds.PeriodEnd <= bounds.PeriodStart {
		return bounds, errortypes.NewValidationError(
			"periodEnd",
			errortypes.ErrInvalid,
			"Period end must be after the period start",
		)
	}
	return bounds, nil
}

type PeriodBounds struct {
	PeriodStart int64 `json:"periodStart"`
	PeriodEnd   int64 `json:"periodEnd"`
	PayDate     int64 `json:"payDate"`
}

// ResolvePeriod computes the current pay period from a settlement control's
// frequency, period-end weekday, and pay delay, in the organization's time
// zone. The period end is exclusive: it is the local midnight after the most
// recent occurrence of the configured end day, so a week closes when the
// organization's day ends, and calendar arithmetic keeps local midnight across
// daylight saving changes.
func ResolvePeriod(
	frequency settlementcontrol.PayPeriodFrequency,
	endDayOfWeek, payDelayDays int,
	now int64,
	loc *time.Location,
) PeriodBounds {
	loc = locationOrUTC(loc)
	nowTime := time.Unix(now, 0).In(loc)
	endDay := time.Weekday(endDayOfWeek)

	daysBack := int(nowTime.Weekday() - endDay)
	if daysBack < 0 {
		daysBack += 7
	}
	periodEndDate := time.Date(
		nowTime.Year(), nowTime.Month(), nowTime.Day()-daysBack+1,
		0, 0, 0, 0, loc,
	)

	periodStartDate := StepPeriods(frequency, periodEndDate, -1)

	return PeriodBounds{
		PeriodStart: periodStartDate.Unix(),
		PeriodEnd:   periodEndDate.Unix(),
		PayDate:     periodEndDate.AddDate(0, 0, payDelayDays).Unix(),
	}
}

func locationOrUTC(loc *time.Location) *time.Location {
	if loc == nil {
		return time.UTC
	}
	return loc
}

func StepPeriods(
	frequency settlementcontrol.PayPeriodFrequency,
	anchor time.Time,
	periods int,
) time.Time {
	switch frequency {
	case settlementcontrol.PayPeriodFrequencyBiweekly:
		return anchor.AddDate(0, 0, 14*periods)
	case settlementcontrol.PayPeriodFrequencyMonthly:
		return timeutils.AddMonths(anchor, periods)
	case settlementcontrol.PayPeriodFrequencyWeekly:
		return anchor.AddDate(0, 0, 7*periods)
	default:
		return anchor.AddDate(0, 0, 7*periods)
	}
}
