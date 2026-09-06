package worker

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenureMonths(t *testing.T) {
	loc := time.UTC
	hire := time.Date(2024, time.March, 15, 0, 0, 0, 0, loc).Unix()

	assert.Equal(t, int32(0), TenureMonths(0, hire, loc))
	assert.Equal(t, int32(0), TenureMonths(hire, hire, loc))
	assert.Equal(t, int32(0), TenureMonths(hire, time.Date(2024, time.April, 14, 0, 0, 0, 0, loc).Unix(), loc))
	assert.Equal(t, int32(1), TenureMonths(hire, time.Date(2024, time.April, 15, 0, 0, 0, 0, loc).Unix(), loc))
	assert.Equal(t, int32(11), TenureMonths(hire, time.Date(2025, time.March, 14, 0, 0, 0, 0, loc).Unix(), loc))
	assert.Equal(t, int32(12), TenureMonths(hire, time.Date(2025, time.March, 15, 0, 0, 0, 0, loc).Unix(), loc))
	assert.Equal(t, int32(24), TenureMonths(hire, time.Date(2026, time.March, 15, 12, 0, 0, 0, loc).Unix(), loc))
}

func TestPTOPolicyRule_Tiers(t *testing.T) {
	rule := PTOPolicyRule{
		PTOType:           PTOTypeVacation,
		AccrualMethod:     PTOAccrualMethodMonthly,
		AccrualAmountDays: decimal.RequireFromString("0.83"),
		MaxBalanceDays:    decimal.NewNullDecimal(decimal.NewFromInt(15)),
		OnTermination:     PTOTerminationPayOut,
		Tiers: []PTOAccrualTier{
			{MinMonths: 12, AccrualAmountDays: decimal.RequireFromString("1.25")},
			{MinMonths: 60, AccrualAmountDays: decimal.RequireFromString("1.67"), MaxBalanceDays: decimal.NewNullDecimal(decimal.NewFromInt(25))},
		},
	}

	assert.True(t, rule.AmountFor(0).Equal(decimal.RequireFromString("0.83")))
	assert.True(t, rule.AmountFor(11).Equal(decimal.RequireFromString("0.83")))
	assert.True(t, rule.AmountFor(12).Equal(decimal.RequireFromString("1.25")))
	assert.True(t, rule.AmountFor(59).Equal(decimal.RequireFromString("1.25")))
	assert.True(t, rule.AmountFor(60).Equal(decimal.RequireFromString("1.67")))
	assert.True(t, rule.MaxBalanceFor(12).Decimal.Equal(decimal.NewFromInt(15)), "a tier without a cap inherits the rule's")
	assert.True(t, rule.MaxBalanceFor(72).Decimal.Equal(decimal.NewFromInt(25)))

	multiErr := errortypes.NewMultiError()
	rule.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())

	bad := rule
	bad.Tiers = []PTOAccrualTier{
		{MinMonths: 24, AccrualAmountDays: decimal.NewFromInt(1)},
		{MinMonths: 12, AccrualAmountDays: decimal.Zero},
	}
	multiErr = errortypes.NewMultiError()
	bad.Validate(multiErr)
	fields := map[string]bool{}
	for _, fieldErr := range multiErr.Errors {
		fields[fieldErr.Field] = true
	}
	assert.True(t, fields["tiers[1].minMonths"], "tiers must ascend")
	assert.True(t, fields["tiers[1].accrualAmountDays"])

	none := rule
	none.AccrualMethod = PTOAccrualMethodNone
	none.AccrualAmountDays = decimal.Zero
	multiErr = errortypes.NewMultiError()
	none.Validate(multiErr)
	found := false
	for _, fieldErr := range multiErr.Errors {
		if fieldErr.Field == "tiers" {
			found = true
		}
	}
	assert.True(t, found, "tiers make no sense on a rule that does not accrue")
}

func TestHolidayCalendar_AndComputePTODays(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	day := func(y int, m time.Month, d int) int64 { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() }

	cal := NewHolidayCalendar([]*OrgHoliday{
		{Name: "Independence Day", HolidayDate: day(2000, time.July, 4), Kind: HolidayKindHoliday, RecursAnnually: true},
		{Name: "Company picnic", HolidayDate: day(2026, time.July, 6), Kind: HolidayKindHoliday},
		{Name: "Peak freeze", HolidayDate: day(2026, time.July, 7), Kind: HolidayKindBlackout},
		{Name: "Christmas Eve freeze", HolidayDate: day(2000, time.December, 24), Kind: HolidayKindBlackout, RecursAnnually: true},
	})
	assert.False(t, cal.IsEmpty())

	assert.True(t, cal.HolidayOn(time.Date(2026, time.July, 4, 0, 0, 0, 0, ny)), "recurring holiday matches any year")
	assert.True(t, cal.HolidayOn(time.Date(2026, time.July, 6, 0, 0, 0, 0, ny)))
	assert.False(t, cal.HolidayOn(time.Date(2027, time.July, 6, 0, 0, 0, 0, ny)), "one-off holidays do not repeat")
	assert.NotNil(t, cal.BlackoutOn(time.Date(2026, time.July, 7, 0, 0, 0, 0, ny)))
	assert.NotNil(t, cal.BlackoutOn(time.Date(2031, time.December, 24, 0, 0, 0, 0, ny)))
	assert.Nil(t, cal.BlackoutOn(time.Date(2026, time.July, 8, 0, 0, 0, 0, ny)))

	// Mon Jun 29 .. Fri Jul 10 2026: 10 weekdays, minus Jul 3 (observed? no — Jul 4 is a Saturday in 2026, so
	// the recurring holiday falls on a weekend and is already skipped) and minus Jul 6 (Mon, one-off holiday).
	start := time.Date(2026, time.June, 29, 0, 0, 0, 0, ny).Unix()
	end := time.Date(2026, time.July, 10, 0, 0, 0, 0, ny).Unix()
	assert.True(t, ComputePTODays(start, end, ny, false, cal).Equal(decimal.NewFromInt(9)))
	assert.True(t, ComputePTODays(start, end, ny, true, cal).Equal(decimal.NewFromInt(12)),
		"counting weekends counts holidays too")
	assert.True(t, ComputePTODays(start, end, ny, false, nil).Equal(decimal.NewFromInt(10)),
		"no calendar, no holidays")

	blackouts := cal.BlackoutsBetween(start, end, ny)
	require.Len(t, blackouts, 1)
	assert.Equal(t, "Peak freeze", blackouts[0].Name)

	december := cal.BlackoutsBetween(
		time.Date(2027, time.December, 20, 0, 0, 0, 0, ny).Unix(),
		time.Date(2027, time.December, 26, 0, 0, 0, 0, ny).Unix(),
		ny,
	)
	require.Len(t, december, 1)
	assert.Equal(t, "Christmas Eve freeze", december[0].Name)
	assert.Empty(t, (*HolidayCalendar)(nil).BlackoutsBetween(start, end, ny))
}

func TestEmploymentEvent_LeaveTypeRoundTrip(t *testing.T) {
	wrk := &Worker{CanBeAssigned: true}
	started := &WorkerEmploymentEvent{
		Kind:     EmploymentEventLeaveStarted,
		ToValues: map[string]string{EmploymentValueLeaveType: "FMLA"},
	}
	assert.True(t, started.Apply(wrk))
	assert.Equal(t, LeaveTypeFMLA, wrk.LeaveType)
	assert.False(t, wrk.CanBeAssigned)

	untyped := &WorkerEmploymentEvent{Kind: EmploymentEventLeaveStarted}
	other := &Worker{}
	untyped.Apply(other)
	assert.Equal(t, LeaveTypeOther, other.LeaveType, "a leave without a type is still a leave")

	ended := &WorkerEmploymentEvent{Kind: EmploymentEventLeaveEnded}
	assert.True(t, ended.Apply(wrk))
	assert.Empty(t, wrk.LeaveType)
	assert.True(t, wrk.CanBeAssigned)

	wrk.LeaveType = LeaveTypeMedical
	(&WorkerEmploymentEvent{Kind: EmploymentEventTerminated}).Apply(wrk)
	assert.Empty(t, wrk.LeaveType, "termination closes any leave")
}
