package workerleave_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func at(year int, month time.Month, day int) int64 {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix()
}

func leaveEntry(on int64, hours float64, counts bool) *workerleave.WorkerLeaveEntry {
	return &workerleave.WorkerLeaveEntry{
		WorkerID:                 pulid.MustNew("wrk_"),
		LeaveCaseID:              pulid.MustNew("wlc_"),
		UsedOn:                   on,
		Hours:                    decimal.NewFromFloat(hours),
		CountsAgainstEntitlement: counts,
	}
}

func TestEntitlementWindow_CalendarYear(t *testing.T) {
	t.Parallel()

	window := workerleave.EntitlementWindow(
		workerleave.MeasureCalendarYear,
		at(2026, time.June, 15),
		at(2020, time.March, 1),
		0,
	)

	assert.Equal(t, at(2026, time.January, 1), window.From)
	assert.Equal(t, at(2027, time.January, 1), window.Through)
}

func TestEntitlementWindow_HireAnniversary(t *testing.T) {
	t.Parallel()

	hire := at(2020, time.March, 10)

	// After this year's anniversary, the period is the one that just started.
	window := workerleave.EntitlementWindow(
		workerleave.MeasureHireAnniversary,
		at(2026, time.June, 15),
		hire,
		0,
	)
	assert.Equal(t, at(2026, time.March, 10), window.From)
	assert.Equal(t, at(2027, time.March, 10), window.Through)

	// Before it, the period is still last year's.
	window = workerleave.EntitlementWindow(
		workerleave.MeasureHireAnniversary,
		at(2026, time.February, 2),
		hire,
		0,
	)
	assert.Equal(t, at(2025, time.March, 10), window.From)
	assert.Equal(t, at(2026, time.March, 10), window.Through)
}

// A leap-day hire has no anniversary in three years out of four. Letting the
// date overflow into March would make the anniversary drift a day whenever the
// leap year came round, so it is clamped to the last day of February — the same
// rule the accrual calendar already uses.
func TestEntitlementWindow_LeapDayAnniversary(t *testing.T) {
	t.Parallel()

	hire := at(2020, time.February, 29)

	window := workerleave.EntitlementWindow(
		workerleave.MeasureHireAnniversary,
		at(2026, time.June, 15),
		hire,
		0,
	)
	assert.Equal(t, at(2026, time.February, 28), window.From)
	assert.Equal(t, at(2027, time.February, 28), window.Through)

	// In a leap year the anniversary falls on the day itself again.
	window = workerleave.EntitlementWindow(
		workerleave.MeasureHireAnniversary,
		at(2028, time.June, 15),
		hire,
		0,
	)
	assert.Equal(t, at(2028, time.February, 29), window.From)
}

// A worker with no hire date would otherwise get a window starting at the
// epoch, which would report fifty years of leave as one period.
func TestEntitlementWindow_HireAnniversaryWithoutAHireDate(t *testing.T) {
	t.Parallel()

	window := workerleave.EntitlementWindow(
		workerleave.MeasureHireAnniversary,
		at(2026, time.June, 15),
		0,
		0,
	)

	assert.Equal(t, at(2026, time.January, 1), window.From)
	assert.Equal(t, at(2027, time.January, 1), window.Through)
}

func TestEntitlementWindow_ForwardFromFirstUse(t *testing.T) {
	t.Parallel()

	firstUse := at(2025, time.April, 1)

	window := workerleave.EntitlementWindow(
		workerleave.MeasureForwardFromFirstUse,
		at(2025, time.September, 1),
		0,
		firstUse,
	)
	assert.Equal(t, firstUse, window.From)
	assert.Equal(t, at(2026, time.April, 1), window.Through)

	// Somebody who first took leave three years ago is in their fourth period,
	// not still in their first.
	window = workerleave.EntitlementWindow(
		workerleave.MeasureForwardFromFirstUse,
		at(2028, time.June, 1),
		0,
		firstUse,
	)
	assert.Equal(t, at(2028, time.April, 1), window.From)
	assert.Equal(t, at(2029, time.April, 1), window.Through)
}

// With nothing taken yet the period has not started, so it starts now.
func TestEntitlementWindow_ForwardWithNoUseYet(t *testing.T) {
	t.Parallel()

	asOf := at(2026, time.June, 15)
	window := workerleave.EntitlementWindow(workerleave.MeasureForwardFromFirstUse, asOf, 0, 0)

	assert.Equal(t, asOf, window.From)
	assert.Equal(t, at(2027, time.June, 15), window.Through)
}

func TestEntitlementWindow_RollingBackward(t *testing.T) {
	t.Parallel()

	asOf := at(2026, time.June, 15)
	window := workerleave.EntitlementWindow(workerleave.MeasureRollingBackward, asOf, 0, 0)

	assert.Equal(t, at(2025, time.June, 15), window.From)
	assert.Equal(t, asOf, window.Through)
}

func TestBuildLeaveEntitlement_CountsOnlyDesignatedDaysInWindow(t *testing.T) {
	t.Parallel()

	asOf := at(2026, time.June, 15)
	in := workerleave.LeaveEntitlementInput{
		Control:  workerleave.DefaultLeaveControl(),
		HireDate: at(2020, time.January, 6),
		AsOf:     asOf,
		Entries: []*workerleave.WorkerLeaveEntry{
			// Inside the rolling year and designated.
			leaveEntry(at(2025, time.August, 4), 8, true),
			leaveEntry(at(2025, time.August, 5), 8, true),
			// Inside the window but the employer chose not to designate it.
			leaveEntry(at(2025, time.September, 1), 8, false),
			// Designated but older than the rolling year.
			leaveEntry(at(2025, time.January, 6), 8, true),
			nil,
		},
	}

	entitlement := workerleave.BuildLeaveEntitlement(in)

	assert.True(t, decimal.NewFromInt(16).Equal(entitlement.UsedHours),
		"used: %s", entitlement.UsedHours)
	assert.True(t, decimal.NewFromInt(480).Equal(entitlement.TotalHours))
	assert.True(t, decimal.NewFromInt(464).Equal(entitlement.RemainingHours))
	assert.True(t, decimal.NewFromInt(12).Equal(entitlement.TotalWeeks))
	assert.False(t, entitlement.Exhausted)
	assert.True(t, entitlement.EligibleOnTenure)
}

// The day being asked about is inside the rolling year: it is the day the
// entitlement is being measured for.
func TestBuildLeaveEntitlement_RollingIncludesToday(t *testing.T) {
	t.Parallel()

	asOf := at(2026, time.June, 15)
	entitlement := workerleave.BuildLeaveEntitlement(workerleave.LeaveEntitlementInput{
		Control: workerleave.DefaultLeaveControl(),
		AsOf:    asOf,
		Entries: []*workerleave.WorkerLeaveEntry{leaveEntry(asOf, 8, true)},
	})

	assert.True(t, decimal.NewFromInt(8).Equal(entitlement.UsedHours))
}

// The end of a fixed period belongs to the next one, or a day would be counted
// against two periods at once.
func TestBuildLeaveEntitlement_FixedPeriodEndIsExclusive(t *testing.T) {
	t.Parallel()

	control := workerleave.DefaultLeaveControl()
	control.MeasurementMethod = workerleave.MeasureCalendarYear

	entitlement := workerleave.BuildLeaveEntitlement(workerleave.LeaveEntitlementInput{
		Control: control,
		AsOf:    at(2026, time.June, 15),
		Entries: []*workerleave.WorkerLeaveEntry{
			leaveEntry(at(2027, time.January, 1), 8, true),
		},
	})

	assert.True(t, entitlement.UsedHours.IsZero())
}

// Designating leave after the fact can push usage past the entitlement. The
// balance floors at zero rather than going negative: nobody is owed back leave.
func TestBuildLeaveEntitlement_OverUseFloorsAtZero(t *testing.T) {
	t.Parallel()

	asOf := at(2026, time.June, 15)
	entries := make([]*workerleave.WorkerLeaveEntry, 0, 70)
	for day := range 70 {
		entries = append(entries, leaveEntry(at(2026, time.January, 1)+int64(day)*86400, 8, true))
	}

	entitlement := workerleave.BuildLeaveEntitlement(workerleave.LeaveEntitlementInput{
		Control: workerleave.DefaultLeaveControl(),
		AsOf:    asOf,
		Entries: entries,
	})

	assert.True(t, decimal.NewFromInt(560).Equal(entitlement.UsedHours))
	assert.True(t, entitlement.RemainingHours.IsZero())
	assert.True(t, entitlement.Exhausted)
}

// A military caregiver case raises the entitlement to 26 weeks for the period.
func TestBuildLeaveEntitlement_MilitaryCaregiver(t *testing.T) {
	t.Parallel()

	entitlement := workerleave.BuildLeaveEntitlement(workerleave.LeaveEntitlementInput{
		Control: workerleave.DefaultLeaveControl(),
		AsOf:    at(2026, time.June, 15),
		Cases: []*workerleave.WorkerLeaveCase{
			{
				Status:            workerleave.LeaveCaseApproved,
				MilitaryCaregiver: true,
				FMLADesignated:    true,
			},
		},
	})

	assert.True(t, decimal.NewFromInt(1040).Equal(entitlement.TotalHours))
	assert.True(t, decimal.NewFromInt(26).Equal(entitlement.TotalWeeks))
	assert.True(t, entitlement.MilitaryCaregiver)
	assert.Equal(t, 1, entitlement.OpenCaseCount)
}

func TestBuildLeaveEntitlement_TenureEligibility(t *testing.T) {
	t.Parallel()

	asOf := at(2026, time.June, 15)

	fresh := workerleave.BuildLeaveEntitlement(workerleave.LeaveEntitlementInput{
		Control:  workerleave.DefaultLeaveControl(),
		HireDate: at(2026, time.March, 1),
		AsOf:     asOf,
	})
	assert.False(t, fresh.EligibleOnTenure)
	assert.Equal(t, 3, fresh.MonthsEmployed)

	settled := workerleave.BuildLeaveEntitlement(workerleave.LeaveEntitlementInput{
		Control:  workerleave.DefaultLeaveControl(),
		HireDate: at(2025, time.June, 15),
		AsOf:     asOf,
	})
	assert.True(t, settled.EligibleOnTenure)
	assert.Equal(t, 12, settled.MonthsEmployed)
}

// A missing control must not read as an entitlement of zero — that would report
// every worker as having exhausted their leave.
func TestBuildLeaveEntitlement_NoControlFallsBackToTheStatute(t *testing.T) {
	t.Parallel()

	entitlement := workerleave.BuildLeaveEntitlement(workerleave.LeaveEntitlementInput{
		AsOf: at(2026, time.June, 15),
	})

	assert.True(t, decimal.NewFromInt(480).Equal(entitlement.TotalHours))
	assert.False(t, entitlement.Exhausted)
	assert.Equal(t, workerleave.MeasureRollingBackward, entitlement.Method)
}

func TestLeaveControl_Validate(t *testing.T) {
	t.Parallel()

	control := workerleave.DefaultLeaveControl()
	multiErr := errortypes.NewMultiError()
	control.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())

	// The statute is a floor: an employer may be more generous but cannot offer
	// less than twelve weeks and call it FMLA.
	stingy := workerleave.DefaultLeaveControl()
	stingy.EntitlementWeeks = decimal.NewFromInt(8)
	multiErr = errortypes.NewMultiError()
	stingy.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "at least twelve weeks")
}

func TestWorkerLeaveCase_Validate(t *testing.T) {
	t.Parallel()

	valid := func() *workerleave.WorkerLeaveCase {
		return &workerleave.WorkerLeaveCase{
			WorkerID:            pulid.MustNew("wrk_"),
			LeaveType:           worker.LeaveTypeFMLA,
			Status:              workerleave.LeaveCasePending,
			Frequency:           workerleave.LeaveContinuous,
			CertificationStatus: workerleave.CertificationNotRequired,
			RequestedAt:         at(2026, time.March, 1),
			StartsAt:            at(2026, time.March, 10),
		}
	}

	multiErr := errortypes.NewMultiError()
	valid().Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())

	t.Run("a decision is a dated act", func(t *testing.T) {
		t.Parallel()
		approved := valid()
		approved.Status = workerleave.LeaveCaseApproved

		multiErr := errortypes.NewMultiError()
		approved.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "when the decision was made")
	})

	t.Run("a denied case cannot be designated", func(t *testing.T) {
		t.Parallel()
		denied := valid()
		denied.Status = workerleave.LeaveCaseDenied
		denied.DecidedAt = ptrInt64(at(2026, time.March, 5))
		denied.FMLADesignated = true

		multiErr := errortypes.NewMultiError()
		denied.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "denied case cannot be designated")
	})

	t.Run("a requested certification records when it was asked for", func(t *testing.T) {
		t.Parallel()
		requested := valid()
		requested.CertificationStatus = workerleave.CertificationRequested

		multiErr := errortypes.NewMultiError()
		requested.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "when the certification was requested")
	})

	t.Run("leave cannot end before it begins", func(t *testing.T) {
		t.Parallel()
		backwards := valid()
		backwards.EndsAt = ptrInt64(at(2026, time.March, 1))

		multiErr := errortypes.NewMultiError()
		backwards.Validate(multiErr)

		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "end before it begins")
	})
}

// A case becomes late simply because the date passed; it should not need a
// sweep to notice.
func TestWorkerLeaveCase_CertificationLate(t *testing.T) {
	t.Parallel()

	due := at(2026, time.March, 20)
	leaveCase := &workerleave.WorkerLeaveCase{
		CertificationStatus:      workerleave.CertificationRequested,
		CertificationRequestedAt: ptrInt64(at(2026, time.March, 5)),
		CertificationDueAt:       &due,
	}

	assert.False(t, leaveCase.CertificationLate(at(2026, time.March, 19)))
	assert.True(t, leaveCase.CertificationLate(at(2026, time.March, 21)))

	// A certification already in hand is never late.
	leaveCase.CertificationStatus = workerleave.CertificationReceived
	assert.False(t, leaveCase.CertificationLate(at(2026, time.March, 21)))
}

func TestWorkerLeaveEntry_Validate(t *testing.T) {
	t.Parallel()

	entry := leaveEntry(at(2026, time.March, 10), 8, true)
	multiErr := errortypes.NewMultiError()
	entry.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())

	tooLong := leaveEntry(at(2026, time.March, 10), 25, true)
	multiErr = errortypes.NewMultiError()
	tooLong.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "more than 24 hours")

	empty := leaveEntry(at(2026, time.March, 10), 0, true)
	multiErr = errortypes.NewMultiError()
	empty.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "more than zero hours")
}
