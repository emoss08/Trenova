package settlementshared

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
)

type PeriodBounds struct {
	PeriodStart int64 `json:"periodStart"`
	PeriodEnd   int64 `json:"periodEnd"`
	PayDate     int64 `json:"payDate"`
}

// ResolvePeriod computes the current pay period from a settlement control's
// frequency, period-end weekday, and pay delay. The period end is exclusive:
// it is the midnight (UTC) after the most recent occurrence of the configured
// end day.
func ResolvePeriod(
	frequency tenant.PayPeriodFrequency,
	endDayOfWeek, payDelayDays int,
	now int64,
) PeriodBounds {
	nowTime := time.Unix(now, 0).UTC()
	endDay := time.Weekday(endDayOfWeek)

	daysBack := int(nowTime.Weekday() - endDay)
	if daysBack < 0 {
		daysBack += 7
	}
	periodEndDate := time.Date(
		nowTime.Year(), nowTime.Month(), nowTime.Day(),
		0, 0, 0, 0, time.UTC,
	).AddDate(0, 0, -daysBack+1)

	var periodStartDate time.Time
	switch frequency {
	case tenant.PayPeriodFrequencyWeekly:
		periodStartDate = periodEndDate.AddDate(0, 0, -7)
	case tenant.PayPeriodFrequencyBiweekly:
		periodStartDate = periodEndDate.AddDate(0, 0, -14)
	case tenant.PayPeriodFrequencyMonthly:
		periodStartDate = periodEndDate.AddDate(0, -1, 0)
	default:
		periodStartDate = periodEndDate.AddDate(0, 0, -7)
	}

	periodEnd := periodEndDate.Unix()
	return PeriodBounds{
		PeriodStart: periodStartDate.Unix(),
		PeriodEnd:   periodEnd,
		PayDate:     periodEndDate.AddDate(0, 0, payDelayDays).Unix(),
	}
}
