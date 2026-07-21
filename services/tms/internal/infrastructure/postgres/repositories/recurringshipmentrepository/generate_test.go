package recurringshipmentrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustUnix(t *testing.T, value, timezone string) int64 {
	t.Helper()

	loc, err := time.LoadLocation(timezone)
	require.NoError(t, err)

	parsed, err := time.ParseInLocation("2006-01-02 15:04", value, loc)
	require.NoError(t, err)

	return parsed.Unix()
}

func activeWeekdaySeries(t *testing.T) *recurringshipment.RecurringShipment {
	t.Helper()

	return &recurringshipment.RecurringShipment{
		ID:              pulid.MustNew("rsh_"),
		Status:          recurringshipment.StatusActive,
		AutoGenerate:    true,
		CronExpression:  "0 8 * * 1-5",
		Timezone:        "America/New_York",
		ExceptionPolicy: recurringshipment.ExceptionPolicySkip,
	}
}

func TestValidateGenerationEligibility(t *testing.T) {
	t.Parallel()

	series := activeWeekdaySeries(t)
	assert.NoError(t, validateGenerationEligibility(series, recurringshipment.RunTriggerAuto))
	assert.NoError(t, validateGenerationEligibility(series, recurringshipment.RunTriggerManual))

	series.Status = recurringshipment.StatusPaused
	assert.Error(t, validateGenerationEligibility(series, recurringshipment.RunTriggerAuto))
	assert.NoError(t, validateGenerationEligibility(series, recurringshipment.RunTriggerManual))

	series.Status = recurringshipment.StatusExpired
	assert.Error(t, validateGenerationEligibility(series, recurringshipment.RunTriggerManual))

	series = activeWeekdaySeries(t)
	limit := int32(2)
	series.MaxOccurrences = &limit
	series.GenerationCount = 2
	assert.Error(t, validateGenerationEligibility(series, recurringshipment.RunTriggerAuto))
}

func TestResolveOccurrence_ExplicitDateWins(t *testing.T) {
	t.Parallel()

	series := activeWeekdaySeries(t)
	stored := mustUnix(t, "2026-07-20 08:00", "America/New_York")
	series.NextOccurrenceAt = &stored

	explicit := mustUnix(t, "2026-07-22 08:00", "America/New_York")
	occurrence, err := resolveOccurrence(series, &repositories.GenerateRecurringShipmentRequest{
		OccurrenceAt: &explicit,
		Trigger:      recurringshipment.RunTriggerManual,
	})
	require.NoError(t, err)
	assert.Equal(t, explicit, occurrence.At)
	assert.Equal(t, explicit, occurrence.OriginalAt)
}

func TestResolveOccurrence_UsesStoredPointerWithShiftMetadata(t *testing.T) {
	t.Parallel()

	series := activeWeekdaySeries(t)
	shifted := mustUnix(t, "2026-07-31 08:00", "America/New_York")
	source := mustUnix(t, "2026-08-01 08:00", "America/New_York")
	series.NextOccurrenceAt = &shifted
	series.NextOccurrenceSourceAt = &source

	occurrence, err := resolveOccurrence(series, &repositories.GenerateRecurringShipmentRequest{
		Trigger: recurringshipment.RunTriggerAuto,
	})
	require.NoError(t, err)
	assert.Equal(t, shifted, occurrence.At)
	assert.Equal(t, source, occurrence.OriginalAt)
	assert.True(t, occurrence.Shifted)
}

func TestAdvanceSeries_AdvancesFromOriginalSlot(t *testing.T) {
	t.Parallel()

	series := &recurringshipment.RecurringShipment{
		Status:          recurringshipment.StatusActive,
		CronExpression:  "30 6 1 * *",
		Timezone:        "America/Chicago",
		SkipWeekends:    true,
		ExceptionPolicy: recurringshipment.ExceptionPolicyPreviousBusinessDay,
	}

	// August 1st 2026 falls on a Saturday, so the occurrence shifted back to
	// Friday July 31st. Advancing must consume the August slot, not re-find it.
	occurrence := &recurringshipment.Occurrence{
		At:         mustUnix(t, "2026-07-31 06:30", "America/Chicago"),
		OriginalAt: mustUnix(t, "2026-08-01 06:30", "America/Chicago"),
		Shifted:    true,
	}

	require.NoError(t, advanceSeries(series, occurrence))
	require.NotNil(t, series.NextOccurrenceAt)
	assert.Equal(
		t,
		mustUnix(t, "2026-09-01 06:30", "America/Chicago"),
		*series.NextOccurrenceAt,
	)
}

func TestAdvanceSeries_ExpiresAtLimit(t *testing.T) {
	t.Parallel()

	series := activeWeekdaySeries(t)
	limit := int32(1)
	series.MaxOccurrences = &limit
	series.GenerationCount = 1

	occurrence := &recurringshipment.Occurrence{
		At:         mustUnix(t, "2026-07-20 08:00", "America/New_York"),
		OriginalAt: mustUnix(t, "2026-07-20 08:00", "America/New_York"),
	}

	require.NoError(t, advanceSeries(series, occurrence))
	assert.Equal(t, recurringshipment.StatusExpired, series.Status)
	assert.Nil(t, series.NextOccurrenceAt)
	assert.Nil(t, series.NextOccurrenceSourceAt)
}

func TestShouldAdvancePointer(t *testing.T) {
	t.Parallel()

	series := activeWeekdaySeries(t)
	next := mustUnix(t, "2026-07-20 08:00", "America/New_York")
	series.NextOccurrenceAt = &next

	occurrence := &recurringshipment.Occurrence{At: next, OriginalAt: next}
	assert.True(t, shouldAdvancePointer(series, &repositories.GenerateRecurringShipmentRequest{
		Trigger: recurringshipment.RunTriggerAuto,
	}, occurrence))
	assert.True(t, shouldAdvancePointer(series, &repositories.GenerateRecurringShipmentRequest{
		Trigger: recurringshipment.RunTriggerManual,
	}, occurrence))

	other := mustUnix(t, "2026-07-24 08:00", "America/New_York")
	assert.False(t, shouldAdvancePointer(series, &repositories.GenerateRecurringShipmentRequest{
		Trigger:      recurringshipment.RunTriggerManual,
		OccurrenceAt: &other,
	}, &recurringshipment.Occurrence{At: other, OriginalAt: other}))
}

func TestDeriveRecurringBOL(t *testing.T) {
	t.Parallel()

	occurrence := mustUnix(t, "2026-07-20 08:00", "America/New_York")

	assert.Equal(t, "ACME-100-20260720", deriveRecurringBOL("ACME-100", occurrence, "America/New_York"))
	assert.Empty(t, deriveRecurringBOL("  ", occurrence, "America/New_York"))

	long := make([]rune, 120)
	for i := range long {
		long[i] = 'A'
	}
	derived := deriveRecurringBOL(string(long), occurrence, "America/New_York")
	assert.LessOrEqual(t, len([]rune(derived)), maxShipmentBOLLength)
	assert.Contains(t, derived, "-20260720")
}

func TestDestinationLocationID_PicksLastDeliveryStop(t *testing.T) {
	t.Parallel()

	firstDelivery := pulid.MustNew("loc_")
	lastDelivery := pulid.MustNew("loc_")
	moves := []*shipment.ShipmentMove{
		{
			Sequence: 0,
			Stops: []*shipment.Stop{
				{Type: shipment.StopTypePickup, Sequence: 0, LocationID: pulid.MustNew("loc_")},
				{Type: shipment.StopTypeDelivery, Sequence: 1, LocationID: firstDelivery},
			},
		},
		{
			Sequence: 1,
			Stops: []*shipment.Stop{
				{Type: shipment.StopTypePickup, Sequence: 0, LocationID: pulid.MustNew("loc_")},
				{Type: shipment.StopTypeDelivery, Sequence: 1, LocationID: lastDelivery},
			},
		},
	}

	assert.Equal(t, lastDelivery, destinationLocationID(moves))
}
