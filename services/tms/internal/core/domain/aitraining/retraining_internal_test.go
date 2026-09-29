package aitraining

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const retrainingNow = int64(1_790_000_000)

func testPolicy() RetrainingPolicy {
	return RetrainingPolicy{
		LookbackDays:         365,
		MinNewExamples:       1000,
		MinIntervalDays:      28,
		RetrainOnDrift:       true,
		MaxPerOrganization:   DefaultMaxPerOrganization,
		ValidationPercent:    DefaultValidationPercent,
		StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
		Gate:                 RetrainingGate{MinAccuracyPercent: 85, MaxRegressionPoints: 0},
	}
}

func daysAgo(days int) int64 {
	return retrainingNow - int64(days)*timeutils.SecondsPerDay
}

func lastCycle(createdDaysAgo int) *RetrainingCycle {
	return &RetrainingCycle{
		Status:     RetrainingStatusPassed,
		CreatedAt:  daysAgo(createdDaysAgo),
		CapturedTo: daysAgo(createdDaysAgo),
	}
}

func TestNewRetrainingWindow(t *testing.T) {
	t.Parallel()

	policy := testPolicy()
	first := NewRetrainingWindow(retrainingNow, &policy, nil)
	assert.Equal(t, daysAgo(365), first.CapturedFrom)
	assert.Equal(t, retrainingNow, first.CapturedTo)
	assert.Equal(t, first.CapturedFrom, first.NewSince, "a first cycle counts the whole window as new")

	after := NewRetrainingWindow(retrainingNow, &policy, lastCycle(30))
	assert.Equal(t, daysAgo(30), after.NewSince, "only corrections after the last window are new")

	ancient := NewRetrainingWindow(retrainingNow, &policy, lastCycle(900))
	assert.Equal(t, ancient.CapturedFrom, ancient.NewSince, "new never starts before the window")

	early := NewRetrainingWindow(1000, &policy, nil)
	assert.Equal(t, int64(0), early.CapturedFrom, "the window never starts before 1970")
}

func TestPlanRetraining(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   RetrainingPlanInput
		status  RetrainingStatus
		reason  RetrainingSkipReason
		trigger RetrainingTrigger
	}{
		{
			name:    "first cycle with enough examples starts",
			input:   RetrainingPlanInput{NewExamples: 1000},
			status:  RetrainingStatusExporting,
			trigger: RetrainingTriggerScheduled,
		},
		{
			name:    "too few new examples",
			input:   RetrainingPlanInput{NewExamples: 999},
			status:  RetrainingStatusSkipped,
			reason:  RetrainingSkipNotEnoughExamples,
			trigger: RetrainingTriggerScheduled,
		},
		{
			name:    "an open cycle wins over everything",
			input:   RetrainingPlanInput{OpenCycle: true, ExportActive: true, NewExamples: 5000},
			status:  RetrainingStatusSkipped,
			reason:  RetrainingSkipCycleOpen,
			trigger: RetrainingTriggerScheduled,
		},
		{
			name:    "another export is running",
			input:   RetrainingPlanInput{ExportActive: true, NewExamples: 5000},
			status:  RetrainingStatusSkipped,
			reason:  RetrainingSkipExportActive,
			trigger: RetrainingTriggerScheduled,
		},
		{
			name:    "the last cycle is too recent",
			input:   RetrainingPlanInput{Last: lastCycle(27), NewExamples: 5000},
			status:  RetrainingStatusSkipped,
			reason:  RetrainingSkipTooSoon,
			trigger: RetrainingTriggerScheduled,
		},
		{
			name:    "exactly the interval has passed",
			input:   RetrainingPlanInput{Last: lastCycle(28), NewExamples: 1000},
			status:  RetrainingStatusExporting,
			trigger: RetrainingTriggerScheduled,
		},
		{
			name:    "drift overrides the interval",
			input:   RetrainingPlanInput{Last: lastCycle(3), NewExamples: 1000, DriftingSeries: 2},
			status:  RetrainingStatusExporting,
			trigger: RetrainingTriggerDrift,
		},
		{
			name:    "drift still needs new examples",
			input:   RetrainingPlanInput{Last: lastCycle(3), NewExamples: 10, DriftingSeries: 1},
			status:  RetrainingStatusSkipped,
			reason:  RetrainingSkipNotEnoughExamples,
			trigger: RetrainingTriggerDrift,
		},
		{
			name:    "manual ignores the interval and the minimum",
			input:   RetrainingPlanInput{Manual: true, Last: lastCycle(1), NewExamples: 0},
			status:  RetrainingStatusExporting,
			trigger: RetrainingTriggerManual,
		},
		{
			name:    "manual still waits for an open cycle",
			input:   RetrainingPlanInput{Manual: true, OpenCycle: true},
			status:  RetrainingStatusSkipped,
			reason:  RetrainingSkipCycleOpen,
			trigger: RetrainingTriggerManual,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := tt.input
			input.Now = retrainingNow
			input.Policy = testPolicy()
			cycle := PlanRetraining(&input)

			assert.Equal(t, tt.status, cycle.Status)
			assert.Equal(t, tt.reason, cycle.SkipReason)
			assert.Equal(t, tt.trigger, cycle.Trigger)
			assert.Equal(t, RetrainingRequestedByScheduler, cycle.RequestedBy)
			if tt.status == RetrainingStatusSkipped {
				require.NotNil(t, cycle.FinishedAt)
			} else {
				assert.Nil(t, cycle.FinishedAt)
			}

			multiErr := errortypes.NewMultiError()
			cycle.Validate(multiErr)
			assert.False(t, multiErr.HasErrors(), multiErr.Error())
		})
	}
}

func TestPlanRetrainingDriftIgnoredWhenTurnedOff(t *testing.T) {
	t.Parallel()

	policy := testPolicy()
	policy.RetrainOnDrift = false
	cycle := PlanRetraining(&RetrainingPlanInput{
		Now:            retrainingNow,
		Policy:         policy,
		Last:           lastCycle(3),
		NewExamples:    5000,
		DriftingSeries: 4,
	})

	assert.Equal(t, RetrainingSkipTooSoon, cycle.SkipReason)
	assert.Equal(t, RetrainingTriggerScheduled, cycle.Trigger)
	assert.Equal(t, 4, cycle.DriftingProviders)
}

func TestPlanRetrainingSnapshotsThePolicy(t *testing.T) {
	t.Parallel()

	policy := testPolicy()
	policy.Gate = RetrainingGate{MinAccuracyPercent: 90, MaxRegressionPoints: 2}
	policy.StructuredOutputMode = aiprovider.StructuredOutputPrompted
	cycle := PlanRetraining(&RetrainingPlanInput{
		Now:         retrainingNow,
		Policy:      policy,
		Manual:      true,
		RequestedBy: "Jordan Lee",
		Note:        "new layouts",
	})

	assert.Equal(t, 90, cycle.MinAccuracyPercent)
	assert.Equal(t, 2, cycle.MaxRegressionPoints)
	assert.Equal(t, aiprovider.StructuredOutputPrompted, cycle.StructuredOutputMode)
	assert.Equal(t, "Jordan Lee", cycle.RequestedBy)
	assert.Equal(t, "new layouts", cycle.Note)
	assert.Equal(t, policy.MinNewExamples, cycle.MinNewExamples)
}

func TestRetrainingCycleExportFinished(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		export  TrainingExport
		changed bool
		status  RetrainingStatus
		message string
	}{
		{
			name:    "still running",
			export:  TrainingExport{Status: ExportStatusRunning},
			changed: false,
			status:  RetrainingStatusExporting,
		},
		{
			name:    "completed with both splits",
			export:  TrainingExport{Status: ExportStatusCompleted, TrainExamples: 900, ValidationExamples: 100},
			changed: true,
			status:  RetrainingStatusReady,
		},
		{
			name:    "completed without a validation set",
			export:  TrainingExport{Status: ExportStatusCompleted, TrainExamples: 12},
			changed: true,
			status:  RetrainingStatusFailed,
			message: "The export kept 12 training and 0 validation examples; both are needed",
		},
		{
			name:    "failed",
			export:  TrainingExport{Status: ExportStatusFailed, FailureMessage: "storage unavailable"},
			changed: true,
			status:  RetrainingStatusFailed,
			message: "The training export was Failed: storage unavailable",
		},
		{
			name:    "canceled",
			export:  TrainingExport{Status: ExportStatusCanceled},
			changed: true,
			status:  RetrainingStatusFailed,
			message: "The training export was Canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cycle := &RetrainingCycle{Status: RetrainingStatusExporting}
			export := tt.export
			assert.Equal(t, tt.changed, cycle.ExportFinished(&export, retrainingNow))
			assert.Equal(t, tt.status, cycle.Status)
			assert.Equal(t, tt.message, cycle.FailureMessage)
			if tt.status == RetrainingStatusFailed {
				require.NotNil(t, cycle.FinishedAt)
			}
		})
	}

	ready := &RetrainingCycle{Status: RetrainingStatusReady}
	assert.False(t, ready.ExportFinished(&TrainingExport{Status: ExportStatusFailed}, retrainingNow),
		"only an exporting cycle follows its export")
}

func TestRetrainingCycleClaimAndLease(t *testing.T) {
	t.Parallel()

	cycle := &RetrainingCycle{ID: pulid.MustNew("airc_"), Status: RetrainingStatusExporting}
	err := cycle.Claim("gpu-1", retrainingNow, 600)
	require.ErrorIs(t, err, ErrRetrainingNotReady)

	cycle.Status = RetrainingStatusReady
	require.NoError(t, cycle.Claim("gpu-1", retrainingNow, 600))
	assert.Equal(t, RetrainingStatusTraining, cycle.Status)
	assert.Equal(t, "gpu-1", cycle.Trainer)
	assert.Equal(t, 1, cycle.Attempts)
	require.NotNil(t, cycle.LeaseExpiresAt)
	assert.Equal(t, retrainingNow+600, *cycle.LeaseExpiresAt)

	assert.False(t, cycle.Claimable(retrainingNow+599), "a held lease cannot be taken")
	require.ErrorIs(t, cycle.Claim("gpu-2", retrainingNow+599, 600), ErrRetrainingNotReady)

	require.NoError(t, cycle.Extend("gpu-1", retrainingNow+500, 600))
	assert.Equal(t, retrainingNow+1100, *cycle.LeaseExpiresAt)
	require.ErrorIs(t, cycle.Extend("gpu-2", retrainingNow+500, 600), ErrRetrainingLeaseLost)

	assert.True(t, cycle.Claimable(retrainingNow+1100), "an expired lease can be taken over")
	require.NoError(t, cycle.Claim("gpu-2", retrainingNow+1100, 600))
	assert.Equal(t, "gpu-2", cycle.Trainer)
	assert.Equal(t, 2, cycle.Attempts)

	require.ErrorIs(t, cycle.Extend("gpu-1", retrainingNow+1200, 600), ErrRetrainingLeaseLost,
		"the trainer that lost its lease learns so on its next heartbeat")
	require.ErrorIs(t, cycle.FailTraining("gpu-1", "boom", retrainingNow), ErrRetrainingLeaseLost)
}

func trainingCycle(gate RetrainingGate) *RetrainingCycle {
	lease := retrainingNow + 600
	return &RetrainingCycle{
		ID:                  pulid.MustNew("airc_"),
		Status:              RetrainingStatusTraining,
		Trainer:             "gpu-1",
		LeaseExpiresAt:      &lease,
		MinAccuracyPercent:  gate.MinAccuracyPercent,
		MaxRegressionPoints: gate.MaxRegressionPoints,
	}
}

func report(modelCorrect, modelScored, baselineCorrect, baselineScored int) *ScoreReport {
	return &ScoreReport{
		Examples: 40,
		Model:    ScoreSide{Correct: modelCorrect, Scored: modelScored},
		Baseline: ScoreSide{Correct: baselineCorrect, Scored: baselineScored},
	}
}

func TestRetrainingCycleSettle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		gate    RetrainingGate
		report  *ScoreReport
		status  RetrainingStatus
		message string
	}{
		{
			name:   "beats production and the minimum",
			gate:   RetrainingGate{MinAccuracyPercent: 85},
			report: report(92, 100, 88, 100),
			status: RetrainingStatusPassed,
		},
		{
			name:   "exactly the minimum accuracy passes",
			gate:   RetrainingGate{MinAccuracyPercent: 85},
			report: report(85, 100, 80, 100),
			status: RetrainingStatusPassed,
		},
		{
			name:    "below the minimum accuracy",
			gate:    RetrainingGate{MinAccuracyPercent: 85},
			report:  report(84, 100, 80, 100),
			status:  RetrainingStatusRejected,
			message: "Accuracy 84.00% is below the minimum of 85%",
		},
		{
			name:    "any regression when none is allowed",
			gate:    RetrainingGate{MinAccuracyPercent: 0, MaxRegressionPoints: 0},
			report:  report(899, 1000, 900, 1000),
			status:  RetrainingStatusRejected,
			message: "Accuracy 89.90% is more than 0 points below production's 90.00%",
		},
		{
			name:   "a regression of exactly the allowance passes",
			gate:   RetrainingGate{MaxRegressionPoints: 5},
			report: report(85, 100, 90, 100),
			status: RetrainingStatusPassed,
		},
		{
			name:    "nothing scored is never a pass",
			gate:    RetrainingGate{},
			report:  report(0, 0, 10, 20),
			status:  RetrainingStatusRejected,
			message: "No field of the validation set was scored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cycle := trainingCycle(tt.gate)
			require.NoError(t, cycle.Settle("gpu-1", &RetrainingResult{
				TrainingConfig: "configs/qwen2.5-7b-instruct.yaml",
				RunDirectory:   "/data/runs/airc",
				ModelDirectory: "/data/runs/airc/dpo/model",
				PromptSHA256:   "abc",
				Report:         tt.report,
			}, retrainingNow))

			assert.Equal(t, tt.status, cycle.Status)
			assert.Equal(t, tt.message, cycle.GateMessage)
			assert.Equal(t, tt.report.Model.Correct, cycle.ModelCorrect)
			assert.Equal(t, tt.report.Baseline.Scored, cycle.BaselineScored)
			assert.Equal(t, "/data/runs/airc/dpo/model", cycle.ModelDirectory)
			assert.Nil(t, cycle.LeaseExpiresAt)
			require.NotNil(t, cycle.FinishedAt)
		})
	}
}

func TestRetrainingCycleSettleNeedsTheLeaseAndAReport(t *testing.T) {
	t.Parallel()

	cycle := trainingCycle(RetrainingGate{})
	require.ErrorIs(t, cycle.Settle("gpu-2", &RetrainingResult{Report: report(1, 1, 1, 1)}, retrainingNow),
		ErrRetrainingLeaseLost)

	var validation *errortypes.Error
	err := cycle.Settle("gpu-1", &RetrainingResult{}, retrainingNow)
	require.Error(t, err)
	assert.True(t, errors.As(err, &validation))
	assert.Equal(t, RetrainingStatusTraining, cycle.Status)
}

func TestRetrainingCycleCancel(t *testing.T) {
	t.Parallel()

	for _, status := range OpenRetrainingStatuses() {
		cycle := &RetrainingCycle{Status: status}
		require.NoError(t, cycle.Cancel(retrainingNow), status)
		assert.Equal(t, RetrainingStatusCanceled, cycle.Status)
		require.NotNil(t, cycle.FinishedAt)
	}

	for _, status := range []RetrainingStatus{
		RetrainingStatusSkipped,
		RetrainingStatusPassed,
		RetrainingStatusRejected,
		RetrainingStatusFailed,
		RetrainingStatusCanceled,
	} {
		cycle := &RetrainingCycle{Status: status}
		require.ErrorIs(t, cycle.Cancel(retrainingNow), ErrRetrainingClosed, status)
		assert.Equal(t, status, cycle.Status)
	}
}

func TestRetrainingCycleValidate(t *testing.T) {
	t.Parallel()

	cycle := PlanRetraining(&RetrainingPlanInput{Now: retrainingNow, Policy: testPolicy(), NewExamples: 5000})
	cycle.SkipReason = RetrainingSkipTooSoon
	cycle.ValidationPercent = 0
	multiErr := errortypes.NewMultiError()
	cycle.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.ElementsMatch(t, []string{"skipReason", "validationPercent"}, fields)
}
