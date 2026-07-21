package recurringshipment

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/shared/cronutils"
)

const (
	maxOccurrenceScanIterations = 400
	maxBusinessDayShiftDays     = 14
)

type Occurrence struct {
	At         int64
	OriginalAt int64
	Shifted    bool
}

// NextOccurrence computes the next generatable occurrence strictly after
// afterUnix, applying the series' weekend/blackout exception policy. It
// returns nil when the series has no further occurrences within its validity
// window (end date or max-occurrence bound already handled by the caller).
func (rs *RecurringShipment) NextOccurrence(afterUnix int64) (*Occurrence, error) {
	loc, err := time.LoadLocation(rs.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load series timezone: %w", err)
	}

	base := afterUnix
	if rs.StartDate > base {
		base = rs.StartDate
	}

	blocked := rs.blockedDates()

	for range maxOccurrenceScanIterations {
		next, nextErr := cronutils.NextRun(rs.CronExpression, rs.Timezone, base)
		if nextErr != nil {
			return nil, nextErr
		}

		if rs.EndDate != nil && next > *rs.EndDate {
			return nil, nil
		}

		occurrence, ok := rs.applyExceptionPolicy(next, loc, blocked)
		if ok {
			return occurrence, nil
		}

		base = next
	}

	return nil, fmt.Errorf(
		"no valid occurrence found within %d schedule iterations",
		maxOccurrenceScanIterations,
	)
}

func (rs *RecurringShipment) applyExceptionPolicy(
	occurrenceAt int64,
	loc *time.Location,
	blocked map[string]struct{},
) (*Occurrence, bool) {
	if !rs.isBlockedDay(occurrenceAt, loc, blocked) {
		return &Occurrence{At: occurrenceAt, OriginalAt: occurrenceAt}, true
	}

	var step time.Duration
	switch rs.ExceptionPolicy {
	case ExceptionPolicyPreviousBusinessDay:
		step = -24 * time.Hour
	case ExceptionPolicyNextBusinessDay:
		step = 24 * time.Hour
	case ExceptionPolicySkip:
		return nil, false
	default:
		return nil, false
	}

	shifted := time.Unix(occurrenceAt, 0).In(loc)
	for range maxBusinessDayShiftDays {
		shifted = shifted.Add(step)
		candidate := shifted.Unix()
		if rs.EndDate != nil && candidate > *rs.EndDate {
			return nil, false
		}

		if !rs.isBlockedDay(candidate, loc, blocked) {
			return &Occurrence{At: candidate, OriginalAt: occurrenceAt, Shifted: true}, true
		}
	}

	return nil, false
}

func (rs *RecurringShipment) isBlockedDay(
	ts int64,
	loc *time.Location,
	blocked map[string]struct{},
) bool {
	local := time.Unix(ts, 0).In(loc)
	if rs.SkipWeekends {
		weekday := local.Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			return true
		}
	}

	_, isBlackout := blocked[local.Format(blackoutDateFmt)]

	return isBlackout
}

func (rs *RecurringShipment) blockedDates() map[string]struct{} {
	if len(rs.BlackoutDates) == 0 {
		return nil
	}

	blocked := make(map[string]struct{}, len(rs.BlackoutDates))
	for _, blackoutDate := range rs.BlackoutDates {
		blocked[blackoutDate] = struct{}{}
	}

	return blocked
}

// GenerationDueAt is the moment a given occurrence becomes eligible for
// materialization, honoring the series' lead time.
func (rs *RecurringShipment) GenerationDueAt(occurrenceAt int64) int64 {
	return occurrenceAt - int64(rs.LeadTimeDays)*int64((24*time.Hour).Seconds())
}

// ReachedOccurrenceLimit reports whether the series has generated its
// configured maximum number of occurrences.
func (rs *RecurringShipment) ReachedOccurrenceLimit() bool {
	return rs.MaxOccurrences != nil && rs.GenerationCount >= int64(*rs.MaxOccurrences)
}
