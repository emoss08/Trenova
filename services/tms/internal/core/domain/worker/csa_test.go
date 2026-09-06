package worker_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func csaAt(year int, month time.Month, day int) int64 {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix()
}

func TestSuggestedBasic(t *testing.T) {
	t.Parallel()

	// A crash is the one event whose BASIC is never in doubt.
	assert.Equal(t,
		worker.BasicCrashIndicator,
		worker.SuggestedBasic(worker.SafetyEventAccident, ""),
	)
	assert.Equal(t,
		worker.BasicUnsafeDriving,
		worker.SuggestedBasic(worker.SafetyEventCitation, ""),
	)
	assert.Equal(t,
		worker.BasicVehicleMaintenance,
		worker.SuggestedBasic(worker.SafetyEventInspection, worker.InspectionResultOutOfService),
	)

	// A clean inspection cites nothing, so it belongs to no BASIC. Filing it
	// under maintenance would make a fleet that passes its inspections look
	// like one that fails them.
	assert.Equal(t,
		worker.CSABasic(""),
		worker.SuggestedBasic(worker.SafetyEventInspection, worker.InspectionResultPass),
	)
	assert.Equal(t, worker.CSABasic(""), worker.SuggestedBasic(worker.SafetyEventNearMiss, ""))
	assert.Equal(t, worker.CSABasic(""), worker.SuggestedBasic(worker.SafetyEventIncident, ""))
}

func TestCSATimeWeight(t *testing.T) {
	t.Parallel()

	now := csaAt(2026, time.September, 4)

	assert.Equal(t, int32(3), worker.CSATimeWeight(csaAt(2026, time.August, 1), now))
	assert.Equal(t, int32(2), worker.CSATimeWeight(csaAt(2026, time.January, 1), now))
	assert.Equal(t, int32(1), worker.CSATimeWeight(csaAt(2025, time.June, 1), now))

	// Past the two-year look-back a violation stops counting entirely — a fleet
	// that cleaned up its act is not the fleet it was.
	assert.Equal(t, int32(0), worker.CSATimeWeight(csaAt(2024, time.June, 1), now))
}

func TestWorkerSafetyViolation_WeightedScore(t *testing.T) {
	t.Parallel()

	now := csaAt(2026, time.September, 4)
	occurred := csaAt(2026, time.August, 1)

	plain := &worker.WorkerSafetyViolation{SeverityWeight: 5}
	assert.Equal(t, int32(15), plain.WeightedScore(occurred, now))

	// Out of service adds two before the recency multiplier, which is the
	// order the FMCSA applies them in.
	oos := &worker.WorkerSafetyViolation{SeverityWeight: 5, OutOfService: true}
	assert.Equal(t, int32(21), oos.WeightedScore(occurred, now))
}

func TestWorkerSafetyViolation_Validate(t *testing.T) {
	t.Parallel()

	t.Run("a weight outside one to ten is refused", func(t *testing.T) {
		t.Parallel()
		entity := &worker.WorkerSafetyViolation{
			SafetyEventID:  "wsev_1",
			WorkerID:       "wrk_1",
			Basic:          worker.BasicUnsafeDriving,
			Description:    "Speeding 15 mph over",
			SeverityWeight: 11,
		}
		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "Severity weight is 1 to 10")
	})

	t.Run("a BASIC outside the seven is refused", func(t *testing.T) {
		t.Parallel()
		entity := &worker.WorkerSafetyViolation{
			SafetyEventID:  "wsev_1",
			WorkerID:       "wrk_1",
			Basic:          worker.CSABasic("Paperwork"),
			Description:    "Something",
			SeverityWeight: 1,
		}
		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "BASIC is not one of the seven")
	})

	t.Run("a valid violation passes", func(t *testing.T) {
		t.Parallel()
		entity := &worker.WorkerSafetyViolation{
			SafetyEventID:  "wsev_1",
			WorkerID:       "wrk_1",
			Basic:          worker.BasicHOSCompliance,
			Code:           "395.8",
			Description:    "No record of duty status",
			SeverityWeight: 5,
		}
		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})
}

func TestBuildFleetBasics_WeightsByRecency(t *testing.T) {
	t.Parallel()

	basics := worker.BuildFleetBasics([]worker.FleetSafetyBasicInput{
		{Basic: worker.BasicHOSCompliance, Bucket: 0, Violations: 2, SeveritySum: 10},
		{Basic: worker.BasicHOSCompliance, Bucket: 2, Violations: 1, SeveritySum: 4},
	}, nil)

	hos := findBasic(t, basics, worker.BasicHOSCompliance)
	assert.Equal(t, int32(3), hos.Violations)
	// 10 × 3 inside six months, plus 4 × 1 out at the look-back.
	assert.Equal(t, int32(34), hos.WeightedScore)
	assert.False(t, hos.Inferred)
}

func TestBuildFleetBasics_OutOfServiceAddsTwoPerViolation(t *testing.T) {
	t.Parallel()

	basics := worker.BuildFleetBasics([]worker.FleetSafetyBasicInput{
		{
			Basic:        worker.BasicVehicleMaintenance,
			Bucket:       1,
			Violations:   2,
			SeveritySum:  6,
			OutOfService: 2,
		},
	}, nil)

	maintenance := findBasic(t, basics, worker.BasicVehicleMaintenance)
	// (6 + 2×2) × 2 — the flag is counted per violation, not per group.
	assert.Equal(t, int32(20), maintenance.WeightedScore)
	assert.Equal(t, int32(2), maintenance.OutOfService)
}

// A carrier who records events but keys no violation codes must still read a
// scorecard, and must be told the figures were inferred rather than cited.
func TestBuildFleetBasics_MarksInferredFigures(t *testing.T) {
	t.Parallel()

	basics := worker.BuildFleetBasics(nil, []worker.FleetSafetyEventBasicInput{
		{Kind: worker.SafetyEventAccident, Bucket: 0, Events: 2, Points: 8},
		{
			Kind:             worker.SafetyEventInspection,
			InspectionResult: worker.InspectionResultPass,
			Bucket:           0,
			Events:           9,
		},
	})

	crash := findBasic(t, basics, worker.BasicCrashIndicator)
	assert.Equal(t, int32(2), crash.Events)
	assert.Equal(t, int32(24), crash.WeightedScore)
	assert.True(t, crash.Inferred)

	// The nine clean inspections belong to no BASIC and must not inflate one.
	maintenance := findBasic(t, basics, worker.BasicVehicleMaintenance)
	assert.Equal(t, int32(0), maintenance.Events)
	assert.False(t, maintenance.Inferred)
}

// Every BASIC comes back in the FMCSA's own order, even the empty ones: a
// scorecard that hides its zeroes reads as a shorter list every month.
func TestBuildFleetBasics_AlwaysReturnsAllSevenInOrder(t *testing.T) {
	t.Parallel()

	basics := worker.BuildFleetBasics(nil, nil)

	require.Len(t, basics, 7)
	for i, expected := range worker.AllCSABasics() {
		assert.Equal(t, expected, basics[i].Basic)
	}
}

func TestAverageScore(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int32(88), worker.AverageScore(880, 10))
	// Rounds to nearest rather than truncating: 87.5 over eight drivers is 88.
	assert.Equal(t, int32(88), worker.AverageScore(700, 8))

	// A fleet with nobody in it does not have a perfect record; it has none.
	assert.Equal(t, int32(0), worker.AverageScore(0, 0))
}

func findBasic(
	t *testing.T,
	basics []worker.FleetSafetyBasic,
	want worker.CSABasic,
) worker.FleetSafetyBasic {
	t.Helper()
	for _, basic := range basics {
		if basic.Basic == want {
			return basic
		}
	}
	t.Fatalf("BASIC %s not present", want)
	return worker.FleetSafetyBasic{}
}
