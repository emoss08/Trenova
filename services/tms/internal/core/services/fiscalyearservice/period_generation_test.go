package fiscalyearservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/shared/pulid"
)

func TestGenerateMonthlyPeriods_CalendarYear(t *testing.T) {
	fy := &fiscalyear.FiscalYear{
		ID:             pulid.MustNew("fy_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		StartDate:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC).Unix(),
	}

	periods := GenerateMonthlyPeriods(fy)

	if len(periods) != 12 {
		t.Fatalf("expected 12 periods, got %d", len(periods))
	}

	expectedMonths := []struct {
		startMonth time.Month
		startDay   int
		endMonth   time.Month
		endDay     int
	}{
		{time.January, 1, time.January, 31},
		{time.February, 1, time.February, 28},
		{time.March, 1, time.March, 31},
		{time.April, 1, time.April, 30},
		{time.May, 1, time.May, 31},
		{time.June, 1, time.June, 30},
		{time.July, 1, time.July, 31},
		{time.August, 1, time.August, 31},
		{time.September, 1, time.September, 30},
		{time.October, 1, time.October, 31},
		{time.November, 1, time.November, 30},
		{time.December, 1, time.December, 31},
	}

	for i, period := range periods {
		startTime := time.Unix(period.StartDate, 0).UTC()
		endTime := time.Unix(period.EndDate, 0).UTC()

		exp := expectedMonths[i]
		if startTime.Month() != exp.startMonth || startTime.Day() != exp.startDay {
			t.Errorf("period %d: expected start %s %d, got %s %d",
				i+1, exp.startMonth, exp.startDay, startTime.Month(), startTime.Day())
		}

		if endTime.Month() != exp.endMonth || endTime.Day() != exp.endDay {
			t.Errorf("period %d: expected end %s %d, got %s %d",
				i+1, exp.endMonth, exp.endDay, endTime.Month(), endTime.Day())
		}

		if period.PeriodNumber != i+1 {
			t.Errorf("period %d: expected period number %d, got %d", i+1, i+1, period.PeriodNumber)
		}

		if period.PeriodType != fiscalperiod.PeriodTypeMonth {
			t.Errorf("period %d: expected type Month, got %s", i+1, period.PeriodType)
		}

		if period.Status != fiscalperiod.StatusOpen {
			t.Errorf("period %d: expected status Open, got %s", i+1, period.Status)
		}

		if period.FiscalYearID != fy.ID {
			t.Errorf("period %d: fiscal year ID mismatch", i+1)
		}
	}
}

func TestGenerateMonthlyPeriods_LeapYear(t *testing.T) {
	fy := &fiscalyear.FiscalYear{
		ID:             pulid.MustNew("fy_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		StartDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC).Unix(),
	}

	periods := GenerateMonthlyPeriods(fy)

	if len(periods) != 12 {
		t.Fatalf("expected 12 periods, got %d", len(periods))
	}

	febEnd := time.Unix(periods[1].EndDate, 0).UTC()
	if febEnd.Day() != 29 {
		t.Errorf("leap year February: expected end day 29, got %d", febEnd.Day())
	}
}

func TestGenerateMonthlyPeriods_NonCalendarFiscalYear(t *testing.T) {
	fy := &fiscalyear.FiscalYear{
		ID:             pulid.MustNew("fy_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		StartDate:      time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC).Unix(),
	}

	periods := GenerateMonthlyPeriods(fy)

	if len(periods) != 12 {
		t.Fatalf("expected 12 periods, got %d", len(periods))
	}

	firstStart := time.Unix(periods[0].StartDate, 0).UTC()
	if firstStart.Month() != time.April || firstStart.Day() != 1 {
		t.Errorf("first period: expected April 1, got %s %d", firstStart.Month(), firstStart.Day())
	}

	lastEnd := time.Unix(periods[11].EndDate, 0).UTC()
	if lastEnd.Month() != time.March || lastEnd.Day() != 31 {
		t.Errorf("last period: expected March 31, got %s %d", lastEnd.Month(), lastEnd.Day())
	}

	for i := 1; i < len(periods); i++ {
		prevEnd := time.Unix(periods[i-1].EndDate, 0).UTC()
		currStart := time.Unix(periods[i].StartDate, 0).UTC()

		gap := currStart.Sub(prevEnd)
		if gap != time.Second {
			t.Errorf("gap between period %d and %d: expected 1s, got %v", i, i+1, gap)
		}
	}
}

func TestGenerateMonthlyPeriods_Contiguous(t *testing.T) {
	fy := &fiscalyear.FiscalYear{
		ID:             pulid.MustNew("fy_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		StartDate:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
		EndDate:        time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC).Unix(),
	}

	periods := GenerateMonthlyPeriods(fy)

	firstStart := time.Unix(periods[0].StartDate, 0).UTC()
	if firstStart.Unix() != fy.StartDate {
		t.Errorf("first period start doesn't match fiscal year start")
	}

	lastEnd := time.Unix(periods[len(periods)-1].EndDate, 0).UTC()
	if lastEnd.Unix() != fy.EndDate {
		t.Errorf("last period end doesn't match fiscal year end")
	}

	for i := 1; i < len(periods); i++ {
		prevEnd := periods[i-1].EndDate
		currStart := periods[i].StartDate
		if currStart-prevEnd != 1 {
			t.Errorf("gap between period %d and %d: expected 1 second, got %d seconds",
				i, i+1, currStart-prevEnd)
		}
	}
}
