package detentionservice_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A backtest is only meaningful because the same settled facts re-priced under
// different terms produce a difference attributable purely to the terms. These
// cases pin the properties the aggregation depends on.

func replay(
	snap detention.PolicySnapshot,
	arrival, departure int64,
	payRate decimal.NullDecimal,
) detentionservice.ComputeResult {
	return detentionservice.Compute(detentionservice.ComputeInput{
		Snapshot:        snap,
		StopType:        shipment.StopTypeDelivery,
		ScheduleType:    shipment.StopScheduleTypeAppointment,
		ArrivedAt:       &arrival,
		DepartedAt:      &departure,
		NoticeSentAt:    &arrival,
		Now:             departure,
		Location:        time.UTC,
		DriverPayRate:   payRate,
		ShipmentAccrued: decimal.Zero,
	})
}

func TestBacktest_ShorterFreeTimeRaisesRevenue(t *testing.T) {
	t.Parallel()

	arrival := baseTime
	departure := baseTime + 4*hour

	current := baseSnapshot()
	proposed := baseSnapshot()
	proposed.FreeMinutes = 90

	before := replay(current, arrival, departure, decimal.NullDecimal{})
	after := replay(proposed, arrival, departure, decimal.NullDecimal{})

	assert.Equal(t, "150.00", before.BillableAmount.StringFixed(2))
	assert.Equal(t, "187.50", after.BillableAmount.StringFixed(2))
	assert.True(t, after.BillableAmount.GreaterThan(before.BillableAmount),
		"cutting free time must increase what the same dwell earns")
}

func TestBacktest_ReplayIsStableAcrossRuns(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	arrival := baseTime
	departure := baseTime + 6*hour

	first := replay(snap, arrival, departure, decimal.NullDecimal{})
	second := replay(snap, arrival, departure, decimal.NullDecimal{})

	assert.Equal(t, first.BillableAmount.String(), second.BillableAmount.String())
	assert.Equal(t, first.RoundedMinutes, second.RoundedMinutes)
	assert.Equal(t, first.Status, second.Status)
}

func TestBacktest_ForfeitTermsCollapseRevenueOnLateStops(t *testing.T) {
	t.Parallel()

	apptEnd := baseTime
	arrival := baseTime + 30*minute
	departure := arrival + 5*hour

	lenient := baseSnapshot()
	strict := baseSnapshot()
	strict.LateArrivalRule = detention.LateArrivalRuleForfeit

	lenientResult := detentionservice.Compute(detentionservice.ComputeInput{
		Snapshot:       lenient,
		StopType:       shipment.StopTypeDelivery,
		ScheduleType:   shipment.StopScheduleTypeAppointment,
		ArrivedAt:      &arrival,
		DepartedAt:     &departure,
		AppointmentEnd: &apptEnd,
		Now:            departure,
		Location:       time.UTC,
	})
	strictResult := detentionservice.Compute(detentionservice.ComputeInput{
		Snapshot:       strict,
		StopType:       shipment.StopTypeDelivery,
		ScheduleType:   shipment.StopScheduleTypeAppointment,
		ArrivedAt:      &arrival,
		DepartedAt:     &departure,
		AppointmentEnd: &apptEnd,
		Now:            departure,
		Location:       time.UTC,
	})

	assert.True(t, lenientResult.BillableAmount.IsPositive())
	assert.True(t, strictResult.BillableAmount.IsZero(),
		"a forfeit clause erases detention revenue on every late arrival")
}

func TestBacktest_NoticeComplianceAssumptionChangesTheAnswer(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.NotificationRequirement = detention.NotificationRequirementRequired
	snap.UnnotifiedBehavior = detention.UnnotifiedBehaviorSuppress
	snap.NotificationLeadMinutes = 30

	arrival := baseTime
	departure := baseTime + 5*hour

	compliant := detentionservice.Compute(detentionservice.ComputeInput{
		Snapshot:     snap,
		StopType:     shipment.StopTypeDelivery,
		ScheduleType: shipment.StopScheduleTypeAppointment,
		ArrivedAt:    &arrival,
		DepartedAt:   &departure,
		NoticeSentAt: &arrival,
		Now:          departure,
		Location:     time.UTC,
	})
	asHappened := detentionservice.Compute(detentionservice.ComputeInput{
		Snapshot:     snap,
		StopType:     shipment.StopTypeDelivery,
		ScheduleType: shipment.StopScheduleTypeAppointment,
		ArrivedAt:    &arrival,
		DepartedAt:   &departure,
		NoticeSentAt: nil,
		Now:          departure,
		Location:     time.UTC,
	})

	assert.Equal(t, "225.00", compliant.BillableAmount.StringFixed(2),
		"assuming compliance measures what the terms are worth")
	assert.True(t, asHappened.BillableAmount.IsZero(),
		"replaying the notices actually sent shows what process failures cost")
	assert.True(t, asHappened.SuppressedByGate)
}

func TestBacktest_MarginGoesNegativeWhenPayFreeTimeIsShorter(t *testing.T) {
	t.Parallel()

	snap := baseSnapshot()
	snap.FreeMinutes = 300
	snap.PayFreeMinutes = 120

	arrival := baseTime
	departure := baseTime + 4*hour

	result := replay(snap, arrival, departure, nullMoney("25.00"))

	require.True(t, result.BillableAmount.IsZero())
	assert.True(t, result.NetMargin.IsNegative(),
		"a customer free-time concession the driver contract does not match loses money")
	assert.Equal(t, "-50.00", result.NetMargin.StringFixed(2))
}

func TestBacktest_TierChangeIsAttributableToTheLadderAlone(t *testing.T) {
	t.Parallel()

	arrival := baseTime
	departure := baseTime + 8*hour

	flat := baseSnapshot()

	tiered := baseSnapshot()
	tiered.RateSource = detention.RateSourceTiers
	tiered.Tiers = []detention.TierSnapshot{
		{FromMinute: 0, ToMinute: ptr(int32(120)), Rate: money("75.00"), RateUnit: detention.TierRateUnitHour},
		{FromMinute: 120, ToMinute: nil, Rate: money("110.00"), RateUnit: detention.TierRateUnitHour},
	}

	flatResult := replay(flat, arrival, departure, decimal.NullDecimal{})
	tieredResult := replay(tiered, arrival, departure, decimal.NullDecimal{})

	assert.Equal(t, flatResult.RoundedMinutes, tieredResult.RoundedMinutes,
		"only the rate changed, so the billable duration must be identical")
	assert.Equal(t, "450.00", flatResult.BillableAmount.StringFixed(2))
	assert.Equal(t, "590.00", tieredResult.BillableAmount.StringFixed(2),
		"2h at 75 plus 4h at 110")
}

func TestBacktest_CapsBoundTheProjection(t *testing.T) {
	t.Parallel()

	arrival := baseTime
	departure := baseTime + 12*hour

	uncapped := baseSnapshot()
	capped := baseSnapshot()
	capped.MaxChargePerStop = nullMoney("250.00")

	uncappedResult := replay(uncapped, arrival, departure, decimal.NullDecimal{})
	cappedResult := replay(capped, arrival, departure, decimal.NullDecimal{})

	assert.True(t, uncappedResult.BillableAmount.GreaterThan(money("250.00")))
	assert.Equal(t, "250.00", cappedResult.BillableAmount.StringFixed(2))
	assert.Equal(t, detention.CapKindPerStopAmount, cappedResult.CapApplied)
}
