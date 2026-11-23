package hosfeasibility_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/hosfeasibility"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

const hourMs = int64(3_600_000)

func TestEvaluate_VerdictFromClocksAndTrip(t *testing.T) {
	t.Parallel()
	now := int64(1_800_000_000)
	state := func(drive int64) *telematics.WorkerHOSState {
		return &telematics.WorkerHOSState{
			WorkerID:         pulid.MustNew("wrk_"),
			DriveRemainingMs: drive,
			ShiftRemainingMs: 14 * hourMs,
			CycleRemainingMs: 60 * hourMs,
			RecordedAt:       now - 60,
		}
	}

	stale := state(10 * hourMs)
	stale.RecordedAt = now - 13*3600
	assert.Equal(t, telematics.FeasibilityVerdictUnknown,
		hosfeasibility.Evaluate(stale, nil, hosfeasibility.Trip{}, now).Verdict)
	assert.Equal(t, telematics.FeasibilityVerdictInfeasible,
		hosfeasibility.Evaluate(state(0), nil, hosfeasibility.Trip{}, now).Verdict)
	assert.Equal(t, telematics.FeasibilityVerdictFeasible,
		hosfeasibility.Evaluate(state(10*hourMs), nil, hosfeasibility.Trip{TotalMiles: 50}, now).Verdict)

	results := []*hosfeasibility.Result{
		{Verdict: telematics.FeasibilityVerdictInfeasible},
		{Verdict: telematics.FeasibilityVerdictFeasible, DriveRemainingMs: 1},
		{Verdict: telematics.FeasibilityVerdictFeasible, DriveRemainingMs: 5},
	}
	hosfeasibility.Sort(results)
	assert.Equal(t, int64(5), results[0].DriveRemainingMs)
	assert.Equal(t, telematics.FeasibilityVerdictInfeasible, results[2].Verdict)
}
