package settlementshared

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/settlementcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chicago(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	return loc
}

func utc(t *testing.T, value string) int64 {
	t.Helper()
	at, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return at.Unix()
}

func TestResolvePeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		frequency settlementcontrol.PayPeriodFrequency
		zone      string
		now       string
		want      PeriodBounds
	}{
		{
			name:      "a Friday evening in Chicago is still Friday, not the UTC Saturday",
			frequency: settlementcontrol.PayPeriodFrequencyWeekly,
			zone:      "America/Chicago",
			now:       "2026-11-14T03:00:00Z",
			want: PeriodBounds{
				PeriodStart: 1_793_509_200,
				PeriodEnd:   1_794_117_600,
				PayDate:     1_794_549_600,
			},
		},
		{
			name:      "the same instant in UTC closes a week later",
			frequency: settlementcontrol.PayPeriodFrequencyWeekly,
			zone:      "UTC",
			now:       "2026-11-14T03:00:00Z",
			want: PeriodBounds{
				PeriodStart: 1_794_614_400 - 7*86_400,
				PeriodEnd:   1_794_614_400 + 86_400,
				PayDate:     1_794_614_400 + 6*86_400,
			},
		},
		{
			name:      "a week that spans the November fall back is 169 hours long",
			frequency: settlementcontrol.PayPeriodFrequencyWeekly,
			zone:      "America/Chicago",
			now:       "2026-11-10T18:00:00Z",
			want: PeriodBounds{
				PeriodStart: 1_793_509_200,
				PeriodEnd:   1_794_117_600,
				PayDate:     1_794_549_600,
			},
		},
		{
			name:      "a week that spans the March spring forward is 167 hours long",
			frequency: settlementcontrol.PayPeriodFrequencyWeekly,
			zone:      "America/Chicago",
			now:       "2026-03-15T12:00:00Z",
			want: PeriodBounds{
				PeriodStart: 1_772_949_600,
				PeriodEnd:   1_773_550_800,
				PayDate:     1_773_982_800,
			},
		},
		{
			name:      "a biweekly period across the spring forward starts at local midnight",
			frequency: settlementcontrol.PayPeriodFrequencyBiweekly,
			zone:      "America/Chicago",
			now:       "2026-03-15T12:00:00Z",
			want: PeriodBounds{
				PeriodStart: 1_772_344_800,
				PeriodEnd:   1_773_550_800,
				PayDate:     1_773_982_800,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loc, err := time.LoadLocation(tt.zone)
			require.NoError(t, err)

			got := ResolvePeriod(tt.frequency, int(time.Saturday), 5, utc(t, tt.now), loc)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolvePeriodBoundariesAreLocalMidnights(t *testing.T) {
	t.Parallel()

	loc := chicago(t)
	got := ResolvePeriod(
		settlementcontrol.PayPeriodFrequencyWeekly,
		int(time.Saturday),
		5,
		utc(t, "2026-11-14T03:00:00Z"),
		loc,
	)

	start := time.Unix(got.PeriodStart, 0).In(loc)
	end := time.Unix(got.PeriodEnd, 0).In(loc)
	pay := time.Unix(got.PayDate, 0).In(loc)

	assert.Equal(t, "2026-11-01T00:00:00-05:00", start.Format(time.RFC3339))
	assert.Equal(t, "2026-11-08T00:00:00-06:00", end.Format(time.RFC3339))
	assert.Equal(t, "2026-11-13T00:00:00-06:00", pay.Format(time.RFC3339))
	assert.Equal(t, 169*time.Hour, end.Sub(start))
}

func TestResolvePeriodWithoutALocationUsesUTC(t *testing.T) {
	t.Parallel()

	now := utc(t, "2026-11-14T03:00:00Z")
	assert.Equal(
		t,
		ResolvePeriod(settlementcontrol.PayPeriodFrequencyWeekly, 6, 5, now, time.UTC),
		ResolvePeriod(settlementcontrol.PayPeriodFrequencyWeekly, 6, 5, now, nil),
	)
}

func TestResolveRequestedPeriodPayDateKeepsLocalMidnightAcrossDST(t *testing.T) {
	t.Parallel()

	loc := chicago(t)
	periodStart := utc(t, "2026-02-28T06:00:00Z")
	periodEnd := utc(t, "2026-03-07T06:00:00Z")

	got, err := ResolveRequestedPeriod(PeriodSchedule{
		Frequency:    settlementcontrol.PayPeriodFrequencyWeekly,
		EndDayOfWeek: int(time.Saturday),
		PayDelayDays: 5,
		Location:     loc,
	}, periodStart, periodEnd, utc(t, "2026-03-20T12:00:00Z"))
	require.NoError(t, err)

	assert.Equal(t, periodStart, got.PeriodStart)
	assert.Equal(t, periodEnd, got.PeriodEnd)
	assert.Equal(
		t,
		"2026-03-12T00:00:00-05:00",
		time.Unix(got.PayDate, 0).In(loc).Format(time.RFC3339),
	)
}

func TestResolveRequestedPeriodResolvesTheCurrentPeriodInTheZone(t *testing.T) {
	t.Parallel()

	loc := chicago(t)
	now := utc(t, "2026-11-14T03:00:00Z")
	schedule := PeriodSchedule{
		Frequency:    settlementcontrol.PayPeriodFrequencyWeekly,
		EndDayOfWeek: int(time.Saturday),
		PayDelayDays: 5,
		Location:     loc,
	}

	got, err := ResolveRequestedPeriod(schedule, 0, 0, now)
	require.NoError(t, err)
	assert.Equal(t, ResolvePeriod(schedule.Frequency, 6, 5, now, loc), got)
}

func TestResolveRequestedPeriodRefusesAnEndBeforeTheStart(t *testing.T) {
	t.Parallel()

	_, err := ResolveRequestedPeriod(PeriodSchedule{Location: chicago(t)}, 200, 100, 0)

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "periodEnd", validationErr.Field())
}

func TestResolvePeriodStepsAMonthByTheCalendar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		lastDay   [3]int
		wantStart string
		wantEnd   string
	}{
		{
			name:      "a period ending 31 March starts 28 February, not 3 March",
			lastDay:   [3]int{2026, 3, 30},
			wantStart: "2026-02-28T00:00:00-06:00",
			wantEnd:   "2026-03-31T00:00:00-05:00",
		},
		{
			name:      "a leap year period ending 31 March starts 29 February",
			lastDay:   [3]int{2024, 3, 30},
			wantStart: "2024-02-29T00:00:00-06:00",
			wantEnd:   "2024-03-31T00:00:00-05:00",
		},
		{
			name:      "a period ending 29 March clamps to 28 February",
			lastDay:   [3]int{2026, 3, 28},
			wantStart: "2026-02-28T00:00:00-06:00",
			wantEnd:   "2026-03-29T00:00:00-05:00",
		},
		{
			name:      "a period ending 31 January starts 31 December",
			lastDay:   [3]int{2026, 1, 30},
			wantStart: "2025-12-31T00:00:00-06:00",
			wantEnd:   "2026-01-31T00:00:00-06:00",
		},
		{
			name:      "a period through 28 February starts 1 February",
			lastDay:   [3]int{2026, 2, 28},
			wantStart: "2026-02-01T00:00:00-06:00",
			wantEnd:   "2026-03-01T00:00:00-06:00",
		},
		{
			name:      "a leap year period through 29 February starts 1 February",
			lastDay:   [3]int{2024, 2, 29},
			wantStart: "2024-02-01T00:00:00-06:00",
			wantEnd:   "2024-03-01T00:00:00-06:00",
		},
		{
			name:      "a leap year period ending 29 February starts 29 January",
			lastDay:   [3]int{2024, 2, 28},
			wantStart: "2024-01-29T00:00:00-06:00",
			wantEnd:   "2024-02-29T00:00:00-06:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loc := chicago(t)
			now := time.Date(
				tt.lastDay[0], time.Month(tt.lastDay[1]), tt.lastDay[2],
				15, 0, 0, 0, loc,
			)
			got := ResolvePeriod(
				settlementcontrol.PayPeriodFrequencyMonthly,
				int(now.Weekday()),
				5,
				now.Unix(),
				loc,
			)

			start := time.Unix(got.PeriodStart, 0).In(loc)
			end := time.Unix(got.PeriodEnd, 0).In(loc)
			assert.Equal(t, tt.wantStart, start.Format(time.RFC3339))
			assert.Equal(t, tt.wantEnd, end.Format(time.RFC3339))
			assert.Equal(t, end.AddDate(0, 0, 5).Unix(), got.PayDate)
		})
	}
}
