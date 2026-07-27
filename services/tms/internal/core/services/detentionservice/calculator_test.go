package detentionservice_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseTime = int64(1767225600)
	hour     = int64(3600)
	minute   = int64(60)
)

func ptr[T any](v T) *T { return &v }

func money(v string) decimal.Decimal {
	d, err := decimal.NewFromString(v)
	if err != nil {
		panic(err)
	}
	return d
}

func nullMoney(v string) decimal.NullDecimal {
	return decimal.NewNullDecimal(money(v))
}

// baseSnapshot is the industry-default contract: 2 hours free, billed hourly at
// $75, rounded up to 15-minute increments.
func baseSnapshot() detention.PolicySnapshot {
	return detention.PolicySnapshot{
		PolicyID:                pulid.MustNew("dtp_"),
		PolicyCode:              "STD-2HR",
		PolicyName:              "Standard 2 Hour Free",
		ClockStartBasis:         detention.ClockStartBasisArrival,
		LateArrivalRule:         detention.LateArrivalRuleNoEffect,
		FreeMinutes:             120,
		PayFreeMinutes:          120,
		BillingIncrementMinutes: 15,
		RoundingMode:            detention.RoundingModeUp,
		RateSource:              detention.RateSourceAccessorial,
		AccessorialCode:         "DET",
		FlatRate:                money("75.00"),
		FlatRateUnit:            detention.TierRateUnitHour,
		DayBoundaryMode:         detention.CapScopePerStop,
		NotificationRequirement: detention.NotificationRequirementNone,
		UnnotifiedBehavior:      detention.UnnotifiedBehaviorBill,
		Currency:                "USD",
	}
}

func baseInput(snap detention.PolicySnapshot) detentionservice.ComputeInput {
	return detentionservice.ComputeInput{
		Snapshot:        snap,
		StopType:        shipment.StopTypeDelivery,
		ScheduleType:    shipment.StopScheduleTypeAppointment,
		ArrivedAt:       ptr(baseTime),
		Now:             baseTime + 10*hour,
		Location:        time.UTC,
		ShipmentAccrued: decimal.Zero,
	}
}

func TestCompute_NoArrival(t *testing.T) {
	t.Parallel()

	in := baseInput(baseSnapshot())
	in.ArrivedAt = nil

	result := detentionservice.Compute(in)

	assert.False(t, result.Applicable)
	assert.True(t, result.BillableAmount.IsZero())
	assert.Equal(t, detention.OccurrenceStatusNotBillable, result.Status)
}

func TestCompute_WithinFreeTime(t *testing.T) {
	t.Parallel()

	in := baseInput(baseSnapshot())
	in.DepartedAt = ptr(baseTime + 90*minute)

	result := detentionservice.Compute(in)

	require.True(t, result.Applicable)
	assert.Equal(t, int32(90), result.RawDwellMinutes)
	assert.Equal(t, int32(0), result.BillableMinutes)
	assert.True(t, result.BillableAmount.IsZero())
	assert.Equal(t, detention.OccurrenceStatusNotBillable, result.Status)
	assert.False(t, result.IsOpen)
}

func TestCompute_ExactlyAtFreeTimeBoundary(t *testing.T) {
	t.Parallel()

	in := baseInput(baseSnapshot())
	in.DepartedAt = ptr(baseTime + 120*minute)

	result := detentionservice.Compute(in)

	assert.Equal(t, int32(120), result.RawDwellMinutes)
	assert.Equal(t, int32(0), result.BillableMinutes,
		"the free-time boundary itself must not be billable")
	assert.True(t, result.BillableAmount.IsZero())
}

func TestCompute_StandardDetention(t *testing.T) {
	t.Parallel()

	in := baseInput(baseSnapshot())
	in.DepartedAt = ptr(baseTime + 5*hour)

	result := detentionservice.Compute(in)

	assert.Equal(t, int32(300), result.RawDwellMinutes)
	assert.Equal(t, int32(180), result.BillableMinutes)
	assert.Equal(t, int32(180), result.RoundedMinutes)
	assert.Equal(t, "225.00", result.BillableAmount.StringFixed(2),
		"3 billable hours at $75/hr")
	assert.Equal(t, detention.OccurrenceStatusPending, result.Status)
}

func TestCompute_OpenStopProjectsToNow(t *testing.T) {
	t.Parallel()

	in := baseInput(baseSnapshot())
	in.DepartedAt = nil
	in.Now = baseTime + 4*hour

	result := detentionservice.Compute(in)

	assert.True(t, result.IsOpen)
	assert.Nil(t, result.ClockStopAt)
	assert.Equal(t, int32(240), result.RawDwellMinutes)
	assert.Equal(t, detention.OccurrenceStatusAccruing, result.Status)
}

func TestCompute_DepartureBeforeArrivalClampsToZero(t *testing.T) {
	t.Parallel()

	in := baseInput(baseSnapshot())
	in.DepartedAt = ptr(baseTime - hour)

	result := detentionservice.Compute(in)

	assert.Equal(t, int32(0), result.RawDwellMinutes,
		"corrupt timestamps must never produce a negative charge")
	assert.True(t, result.BillableAmount.IsZero())
}

func TestCompute_ClockStartBasis(t *testing.T) {
	t.Parallel()

	appointment := baseTime + 2*hour

	tests := []struct {
		name           string
		basis          detention.ClockStartBasis
		arrivedAt      int64
		wantClockStart int64
	}{
		{
			name:           "arrival basis ignores the appointment",
			basis:          detention.ClockStartBasisArrival,
			arrivedAt:      baseTime,
			wantClockStart: baseTime,
		},
		{
			name:           "appointment basis ignores an early arrival",
			basis:          detention.ClockStartBasisAppointment,
			arrivedAt:      baseTime,
			wantClockStart: appointment,
		},
		{
			name:           "later-of picks the appointment when the driver is early",
			basis:          detention.ClockStartBasisLaterOfArrivalOrAppointment,
			arrivedAt:      baseTime,
			wantClockStart: appointment,
		},
		{
			name:           "later-of picks arrival when the driver is on time",
			basis:          detention.ClockStartBasisLaterOfArrivalOrAppointment,
			arrivedAt:      appointment + 10*minute,
			wantClockStart: appointment + 10*minute,
		},
		{
			name:           "earlier-of picks the appointment when it precedes arrival",
			basis:          detention.ClockStartBasisEarlierOfArrivalOrAppointment,
			arrivedAt:      appointment + 30*minute,
			wantClockStart: appointment,
		},
		{
			name:           "earlier-of picks arrival when the driver is early",
			basis:          detention.ClockStartBasisEarlierOfArrivalOrAppointment,
			arrivedAt:      baseTime,
			wantClockStart: baseTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			snap := baseSnapshot()
			snap.ClockStartBasis = tt.basis

			in := baseInput(snap)
			in.ArrivedAt = ptr(tt.arrivedAt)
			in.AppointmentStart = ptr(appointment)
			in.AppointmentEnd = ptr(appointment + hour)
			in.DepartedAt = ptr(appointment + 8*hour)

			result := detentionservice.Compute(in)

			assert.Equal(t, tt.wantClockStart, result.ClockStartAt)
		})
	}
}

func TestCompute_LateArrivalRules(t *testing.T) {
	t.Parallel()

	apptStart := baseTime
	apptEnd := baseTime + hour
	arrival := apptEnd + 30*minute
	departure := arrival + 4*hour

	tests := []struct {
		name            string
		rule            detention.LateArrivalRule
		grace           int16
		wantLate        bool
		wantBillableMin int32
		wantStatus      detention.OccurrenceStatus
	}{
		{
			name:            "NoEffect keeps full entitlement",
			rule:            detention.LateArrivalRuleNoEffect,
			wantLate:        true,
			wantBillableMin: 120,
			wantStatus:      detention.OccurrenceStatusPending,
		},
		{
			name:            "Forfeit voids the charge entirely",
			rule:            detention.LateArrivalRuleForfeit,
			wantLate:        true,
			wantBillableMin: 0,
			wantStatus:      detention.OccurrenceStatusNotBillable,
		},
		{
			name:            "ReduceFreeTime subtracts the lateness",
			rule:            detention.LateArrivalRuleReduceFreeTime,
			wantLate:        true,
			wantBillableMin: 150,
			wantStatus:      detention.OccurrenceStatusPending,
		},
		{
			name:            "grace period suppresses the late finding",
			rule:            detention.LateArrivalRuleForfeit,
			grace:           45,
			wantLate:        false,
			wantBillableMin: 120,
			wantStatus:      detention.OccurrenceStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			snap := baseSnapshot()
			snap.LateArrivalRule = tt.rule
			snap.LateArrivalGraceMinutes = tt.grace

			in := baseInput(snap)
			in.ArrivedAt = ptr(arrival)
			in.AppointmentStart = ptr(apptStart)
			in.AppointmentEnd = ptr(apptEnd)
			in.DepartedAt = ptr(departure)

			result := detentionservice.Compute(in)

			assert.Equal(t, tt.wantLate, result.ArrivedLate)
			assert.Equal(t, tt.wantBillableMin, result.BillableMinutes)
			assert.Equal(t, tt.wantStatus, result.Status)
		})
	}
}

func TestCompute_LateArrivalClockFromAppointment(t *testing.T) {
	t.Parallel()

	apptStart := baseTime
	arrival := baseTime + 90*minute
	departure := arrival + 3*hour

	snap := baseSnapshot()
	snap.LateArrivalRule = detention.LateArrivalRuleClockFromAppointment

	in := baseInput(snap)
	in.ArrivedAt = ptr(arrival)
	in.AppointmentStart = ptr(apptStart)
	in.AppointmentEnd = ptr(apptStart + 30*minute)
	in.DepartedAt = ptr(departure)

	result := detentionservice.Compute(in)

	assert.True(t, result.ArrivedLate)
	assert.Equal(t, apptStart, result.ClockStartAt,
		"the clock anchors to the appointment so lateness burns the carrier's free time")
	assert.Equal(t, int32(270), result.RawDwellMinutes)
	assert.Equal(t, int32(150), result.BillableMinutes)
}

func TestCompute_ReduceFreeTimeCannotGoNegative(t *testing.T) {
	t.Parallel()

	apptEnd := baseTime
	arrival := baseTime + 5*hour

	snap := baseSnapshot()
	snap.LateArrivalRule = detention.LateArrivalRuleReduceFreeTime

	in := baseInput(snap)
	in.ArrivedAt = ptr(arrival)
	in.AppointmentStart = ptr(baseTime - hour)
	in.AppointmentEnd = ptr(apptEnd)
	in.DepartedAt = ptr(arrival + 2*hour)

	result := detentionservice.Compute(in)

	assert.Equal(t, int32(0), result.FreeMinutesGranted,
		"lateness larger than the allowance floors free time at zero, never negative")
	assert.Equal(t, int32(120), result.BillableMinutes)
}

func TestCompute_Rounding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mode      detention.RoundingMode
		increment int16
		dwellMins int64
		wantMins  int32
	}{
		{"up rounds a partial increment", detention.RoundingModeUp, 15, 130, 15},
		{"up leaves an exact increment alone", detention.RoundingModeUp, 15, 135, 15},
		{"down discards the partial increment", detention.RoundingModeDown, 15, 134, 0},
		{"down keeps whole increments", detention.RoundingModeDown, 15, 150, 30},
		{"nearest rounds up at the half", detention.RoundingModeNearest, 60, 150, 60},
		{"nearest rounds down below the half", detention.RoundingModeNearest, 60, 149, 0},
		{"exact bills to the minute", detention.RoundingModeExact, 0, 137, 17},
		{"hourly increment rounds 2h10m to 3h", detention.RoundingModeUp, 60, 190, 120},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			snap := baseSnapshot()
			snap.RoundingMode = tt.mode
			snap.BillingIncrementMinutes = tt.increment

			in := baseInput(snap)
			in.DepartedAt = ptr(baseTime + tt.dwellMins*minute)

			result := detentionservice.Compute(in)

			assert.Equal(t, tt.wantMins, result.RoundedMinutes)
		})
	}
}

func TestCompute_MinimumBillableThreshold(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.MinimumBillableMinutes = 30
	snap.RoundingMode = detention.RoundingModeExact
	snap.BillingIncrementMinutes = 0

	t.Run("below the minimum bills nothing", func(t *testing.T) {
		t.Parallel()

		in := baseInput(snap)
		in.DepartedAt = ptr(baseTime + 140*minute)

		result := detentionservice.Compute(in)

		assert.Equal(t, int32(0), result.BillableMinutes)
		assert.True(t, result.BillableAmount.IsZero())
	})

	t.Run("at the minimum bills in full", func(t *testing.T) {
		t.Parallel()

		in := baseInput(snap)
		in.DepartedAt = ptr(baseTime + 150*minute)

		result := detentionservice.Compute(in)

		assert.Equal(t, int32(30), result.BillableMinutes)
		assert.Equal(t, "37.50", result.BillableAmount.StringFixed(2))
	})
}

func TestCompute_TieredRates(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.RateSource = detention.RateSourceTiers
	snap.RoundingMode = detention.RoundingModeExact
	snap.BillingIncrementMinutes = 0
	snap.Tiers = []detention.TierSnapshot{
		{
			FromMinute: 0,
			ToMinute:   ptr(int32(120)),
			Rate:       money("75.00"),
			RateUnit:   detention.TierRateUnitHour,
			Label:      "First 2 hours",
		},
		{
			FromMinute: 120,
			ToMinute:   nil,
			Rate:       money("100.00"),
			RateUnit:   detention.TierRateUnitHour,
			Label:      "Beyond 2 hours",
		},
	}

	t.Run("charge stays inside the first tier", func(t *testing.T) {
		t.Parallel()

		in := baseInput(snap)
		in.DepartedAt = ptr(baseTime + 3*hour)

		result := detentionservice.Compute(in)

		assert.Equal(t, int32(60), result.RoundedMinutes)
		assert.Equal(t, "75.00", result.BillableAmount.StringFixed(2))
	})

	t.Run("charge escalates across the tier boundary", func(t *testing.T) {
		t.Parallel()

		in := baseInput(snap)
		in.DepartedAt = ptr(baseTime + 6*hour)

		result := detentionservice.Compute(in)

		assert.Equal(t, int32(240), result.RoundedMinutes)
		assert.Equal(t, "350.00", result.BillableAmount.StringFixed(2),
			"2h at $75 plus 2h at $100")
	})

	t.Run("exactly at the boundary uses only the first tier", func(t *testing.T) {
		t.Parallel()

		in := baseInput(snap)
		in.DepartedAt = ptr(baseTime + 4*hour)

		result := detentionservice.Compute(in)

		assert.Equal(t, int32(120), result.RoundedMinutes)
		assert.Equal(t, "150.00", result.BillableAmount.StringFixed(2),
			"the half-open band must not spill into the escalated tier")
	})
}

func TestCompute_FlatTierUnit(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.RateSource = detention.RateSourceTiers
	snap.Tiers = []detention.TierSnapshot{
		{
			FromMinute: 0,
			ToMinute:   nil,
			Rate:       money("250.00"),
			RateUnit:   detention.TierRateUnitFlat,
			Label:      "Flat detention fee",
		},
	}

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 9*hour)

	result := detentionservice.Compute(in)

	assert.Equal(t, "250.00", result.BillableAmount.StringFixed(2),
		"a flat tier charges once regardless of duration")
}

func TestCompute_MaxBillableMinutesCap(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.MaxBillableMinutesPerStop = ptr(int32(240))

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 12*hour)

	result := detentionservice.Compute(in)

	assert.Equal(t, int32(240), result.RoundedMinutes)
	assert.Equal(t, detention.CapKindMaxMinutes, result.CapApplied)
	assert.Equal(t, "300.00", result.BillableAmount.StringFixed(2))
}

func TestCompute_PerStopAmountCap(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.MaxChargePerStop = nullMoney("200.00")

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 8*hour)

	result := detentionservice.Compute(in)

	assert.Equal(t, "450.00", result.GrossAmount.StringFixed(2))
	assert.Equal(t, "200.00", result.BillableAmount.StringFixed(2))
	assert.Equal(t, detention.CapKindPerStopAmount, result.CapApplied)
}

// A stay spanning midnight is the classic daily-cap trap: naive engines apply
// the cap twice and overbill, or once and underbill.
func TestCompute_PerDayCapAcrossMidnight(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	arrival := time.Date(2026, 1, 1, 18, 0, 0, 0, loc).Unix()
	departure := time.Date(2026, 1, 2, 8, 0, 0, 0, loc).Unix()

	snap := baseSnapshot()
	snap.FreeMinutes = 120
	snap.MaxChargePerDay = nullMoney("300.00")

	in := baseInput(snap)
	in.ArrivedAt = ptr(arrival)
	in.DepartedAt = ptr(departure)
	in.Location = loc
	in.Now = departure

	result := detentionservice.Compute(in)

	assert.Equal(t, int32(840), result.RawDwellMinutes)
	assert.Equal(t, int32(720), result.RoundedMinutes)
	assert.Equal(t, "900.00", result.GrossAmount.StringFixed(2))
	assert.Len(t, result.DayAllocation, 2,
		"the billable window touches two calendar days")
	assert.Equal(t, detention.CapKindPerDayAmount, result.CapApplied)
	assert.Equal(t, "600.00", result.BillableAmount.StringFixed(2),
		"each calendar day is capped independently at $300")
}

func TestCompute_PerDayCapSingleDay(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.MaxChargePerDay = nullMoney("300.00")

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 10*hour)

	result := detentionservice.Compute(in)

	assert.Len(t, result.DayAllocation, 1)
	assert.Equal(t, "300.00", result.BillableAmount.StringFixed(2))
	assert.Equal(t, detention.CapKindPerDayAmount, result.CapApplied)
}

func TestCompute_PerDayCapRespectsPriorAccrual(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.MaxChargePerDay = nullMoney("300.00")

	dayKey := time.Unix(baseTime, 0).UTC().Format(time.DateOnly)

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 6*hour)
	in.DayAccrued = map[string]decimal.Decimal{dayKey: money("250.00")}

	result := detentionservice.Compute(in)

	assert.Equal(t, "50.00", result.BillableAmount.StringFixed(2),
		"only the unused remainder of the daily cap is available")
}

func TestCompute_PerShipmentCap(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.MaxChargePerShipment = nullMoney("400.00")

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 8*hour)
	in.ShipmentAccrued = money("325.00")

	result := detentionservice.Compute(in)

	assert.Equal(t, "75.00", result.BillableAmount.StringFixed(2))
	assert.Equal(t, detention.CapKindPerShipment, result.CapApplied)
}

func TestCompute_PerShipmentCapAlreadyExhausted(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.MaxChargePerShipment = nullMoney("400.00")

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 8*hour)
	in.ShipmentAccrued = money("500.00")

	result := detentionservice.Compute(in)

	assert.True(t, result.BillableAmount.IsZero(),
		"an over-accrued shipment cap must not produce a negative charge")
	assert.Equal(t, detention.OccurrenceStatusNotBillable, result.Status)
}

func TestCompute_LayoverConversion(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.ConvertToLayoverAtMinutes = ptr(int32(1440))
	snap.LayoverAccessorialChargeID = ptr(pulid.MustNew("acc_"))

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 30*hour)
	in.Now = baseTime + 30*hour

	result := detentionservice.Compute(in)

	assert.True(t, result.ConvertedToLayover)
	assert.Equal(t, int32(1320), result.RoundedMinutes,
		"detention accrues only up to the 24h layover boundary minus free time")
	assert.Equal(t, detention.CapKindLayoverBoundary, result.CapApplied)
}

func TestCompute_NotificationGate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		behavior       detention.UnnotifiedBehavior
		noticeSentAt   *int64
		wantStatus     detention.NotificationStatus
		wantSuppressed bool
		wantApproval   bool
		wantAmount     string
	}{
		{
			name:         "notice sent in time bills normally",
			behavior:     detention.UnnotifiedBehaviorSuppress,
			noticeSentAt: ptr(baseTime + 100*minute),
			wantStatus:   detention.NotificationStatusSent,
			wantAmount:   "225.00",
		},
		{
			name:           "missed notice suppresses the charge",
			behavior:       detention.UnnotifiedBehaviorSuppress,
			noticeSentAt:   nil,
			wantStatus:     detention.NotificationStatusMissed,
			wantSuppressed: true,
			wantAmount:     "0.00",
		},
		{
			name:         "missed notice flags for review",
			behavior:     detention.UnnotifiedBehaviorFlag,
			noticeSentAt: nil,
			wantStatus:   detention.NotificationStatusMissed,
			wantApproval: true,
			wantAmount:   "225.00",
		},
		{
			name:         "late notice is not a satisfied requirement",
			behavior:     detention.UnnotifiedBehaviorFlag,
			noticeSentAt: ptr(baseTime + 300*minute),
			wantStatus:   detention.NotificationStatusLate,
			wantApproval: true,
			wantAmount:   "225.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			snap := baseSnapshot()
			snap.NotificationRequirement = detention.NotificationRequirementRequired
			snap.NotificationLeadMinutes = 30
			snap.NotificationDeadlineMinutes = 0
			snap.UnnotifiedBehavior = tt.behavior

			in := baseInput(snap)
			in.DepartedAt = ptr(baseTime + 5*hour)
			in.NoticeSentAt = tt.noticeSentAt

			result := detentionservice.Compute(in)

			assert.Equal(t, tt.wantStatus, result.NotificationStatus)
			assert.Equal(t, tt.wantSuppressed, result.SuppressedByGate)
			assert.Equal(t, tt.wantApproval, result.RequiresApproval)
			assert.Equal(t, tt.wantAmount, result.BillableAmount.StringFixed(2))
		})
	}
}

func TestCompute_NoticeDeadlineTimestamps(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.NotificationRequirement = detention.NotificationRequirementRequired
	snap.NotificationLeadMinutes = 30
	snap.NotificationDeadlineMinutes = 15

	in := baseInput(snap)
	in.DepartedAt = nil
	in.Now = baseTime + 60*minute

	result := detentionservice.Compute(in)

	require.NotNil(t, result.NoticeDueAt)
	require.NotNil(t, result.NoticeDeadlineAt)
	assert.Equal(t, baseTime+90*minute, *result.NoticeDueAt,
		"the warning fires 30 min before free time expires")
	assert.Equal(t, baseTime+135*minute, *result.NoticeDeadlineAt,
		"the contractual deadline is 15 min after expiry")
	assert.Equal(t, detention.NotificationStatusPending, result.NotificationStatus)
}

func TestCompute_OpenOccurrencePendingNoticeIsNotGated(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.NotificationRequirement = detention.NotificationRequirementRequired
	snap.NotificationDeadlineMinutes = 60
	snap.UnnotifiedBehavior = detention.UnnotifiedBehaviorSuppress

	in := baseInput(snap)
	in.DepartedAt = nil
	in.Now = baseTime + 150*minute

	result := detentionservice.Compute(in)

	assert.Equal(t, detention.NotificationStatusPending, result.NotificationStatus,
		"the deadline has not passed, so dispatch can still save the claim")
	assert.False(t, result.SuppressedByGate,
		"a still-open notice window must not suppress an accruing charge")
	assert.Equal(t, detention.OccurrenceStatusAccruing, result.Status)
	assert.Equal(t, "37.50", result.BillableAmount.StringFixed(2))
}

// Once the notice deadline lapses the claim is already lost, even though the
// driver is still on site. Surfacing that while it is happening is the point.
func TestCompute_OpenOccurrenceWithMissedDeadlineIsSuppressed(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.NotificationRequirement = detention.NotificationRequirementRequired
	snap.NotificationDeadlineMinutes = 0
	snap.UnnotifiedBehavior = detention.UnnotifiedBehaviorSuppress

	in := baseInput(snap)
	in.DepartedAt = nil
	in.Now = baseTime + 150*minute

	result := detentionservice.Compute(in)

	assert.Equal(t, detention.NotificationStatusMissed, result.NotificationStatus)
	assert.True(t, result.SuppressedByGate)
	assert.True(t, result.BillableAmount.IsZero(),
		"no notice inside the contractual window means nothing is collectable")
	assert.Equal(t, detention.OccurrenceStatusAccruing, result.Status,
		"the driver is still on site, so the occurrence keeps accruing time")
}

func TestCompute_DriverPayAndMargin(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.PayFreeMinutes = 120

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 5*hour)
	in.DriverPayRate = nullMoney("25.00")

	result := detentionservice.Compute(in)

	assert.Equal(t, int32(180), result.DriverPayMinutes)
	assert.Equal(t, "75.00", result.DriverPayAmount.StringFixed(2))
	assert.Equal(t, "225.00", result.BillableAmount.StringFixed(2))
	assert.Equal(t, "150.00", result.NetMargin.StringFixed(2))
}

// The margin leak the AR/AP view exists to expose: a customer negotiated four
// hours of free time but the driver is still paid from two.
func TestCompute_AsymmetricFreeTimeLeaksMargin(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.FreeMinutes = 240
	snap.PayFreeMinutes = 120

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 4*hour)
	in.DriverPayRate = nullMoney("25.00")

	result := detentionservice.Compute(in)

	assert.True(t, result.BillableAmount.IsZero(),
		"nothing is billable inside the customer's four free hours")
	assert.Equal(t, int32(120), result.DriverPayMinutes)
	assert.Equal(t, "50.00", result.DriverPayAmount.StringFixed(2))
	assert.Equal(t, "-50.00", result.NetMargin.StringFixed(2),
		"the carrier pays the driver for time it cannot bill")
}

func TestCompute_AutoApproveUnderThreshold(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.AutoApproveUnderAmount = nullMoney("300.00")

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 4*hour)

	result := detentionservice.Compute(in)

	assert.Equal(t, "150.00", result.BillableAmount.StringFixed(2))
	assert.Equal(t, detention.OccurrenceStatusApproved, result.Status)
}

func TestCompute_RequireApprovalOverThreshold(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.RequireApprovalOverAmount = nullMoney("200.00")
	snap.AutoApproveUnderAmount = nullMoney("100.00")

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 6*hour)

	result := detentionservice.Compute(in)

	assert.True(t, result.RequiresApproval)
	assert.Equal(t, detention.OccurrenceStatusPending, result.Status)
}

func TestCompute_StopTypeFreeTimeOverrideIsHonoredViaSnapshot(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.FreeMinutes = 60

	in := baseInput(snap)
	in.StopType = shipment.StopTypePickup
	in.DepartedAt = ptr(baseTime + 3*hour)

	result := detentionservice.Compute(in)

	assert.Equal(t, int32(60), result.FreeMinutesGranted)
	assert.Equal(t, int32(120), result.BillableMinutes)
}

func TestCompute_IsDeterministic(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.RateSource = detention.RateSourceTiers
	snap.Tiers = []detention.TierSnapshot{
		{FromMinute: 0, ToMinute: ptr(int32(60)), Rate: money("75.00"), RateUnit: detention.TierRateUnitHour},
		{FromMinute: 60, ToMinute: nil, Rate: money("110.00"), RateUnit: detention.TierRateUnitHour},
	}
	snap.MaxChargePerDay = nullMoney("500.00")

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 7*hour)
	in.DriverPayRate = nullMoney("25.00")

	first := detentionservice.Compute(in)
	second := detentionservice.Compute(in)

	assert.Equal(t, first.BillableAmount.String(), second.BillableAmount.String())
	assert.Equal(t, first.RoundedMinutes, second.RoundedMinutes)
	assert.Equal(t, first.DriverPayAmount.String(), second.DriverPayAmount.String())
	assert.Equal(t, len(first.Trace.Steps), len(second.Trace.Steps),
		"replaying identical inputs must reproduce an identical derivation")
}

func TestCompute_TraceIsPopulatedAndRenderable(t *testing.T) {
	t.Parallel()

	in := baseInput(baseSnapshot())
	in.DepartedAt = ptr(baseTime + 5*hour)

	result := detentionservice.Compute(in)

	require.NotNil(t, result.Trace)
	assert.NotEmpty(t, result.Trace.Steps)
	assert.Equal(t, detention.EngineVersion, result.Trace.EngineVer)

	receipt := result.Trace.Receipt()
	assert.Contains(t, receipt, "Clock started")
	assert.Contains(t, receipt, "Free time deducted")
	assert.Contains(t, receipt, "Billable detention")
}

func TestCompute_CapOrderingStopBeforeShipment(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.MaxChargePerStop = nullMoney("300.00")
	snap.MaxChargePerShipment = nullMoney("400.00")

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 10*hour)
	in.ShipmentAccrued = money("250.00")

	result := detentionservice.Compute(in)

	assert.Equal(t, "150.00", result.BillableAmount.StringFixed(2),
		"the stop cap clamps to 300, then the shipment cap leaves only 150")
	assert.Equal(t, detention.CapKindPerShipment, result.CapApplied)
}

func TestCompute_ZeroRateProducesZeroCharge(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.FlatRate = decimal.Zero

	in := baseInput(snap)
	in.DepartedAt = ptr(baseTime + 6*hour)

	result := detentionservice.Compute(in)

	assert.True(t, result.BillableAmount.IsZero())
	assert.Equal(t, detention.OccurrenceStatusNotBillable, result.Status)
	assert.Equal(t, int32(240), result.RoundedMinutes,
		"minutes are still recorded even when the rate is zero")
}
