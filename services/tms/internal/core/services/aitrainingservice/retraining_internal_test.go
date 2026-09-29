package aitrainingservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const day = timeutils.SecondsPerDay

func scheduled() *services.PlanAIRetrainingRequest {
	return &services.PlanAIRetrainingRequest{}
}

func TestRetrainerPlanDoesNothingWhileDisabled(t *testing.T) {
	t.Parallel()

	f := newRetrainingFixture()
	f.cfg.AIRetraining.Enabled = false
	f.corrections.trainable = 50_000

	cycle, err := f.service.Plan(t.Context(), scheduled())
	require.NoError(t, err)
	assert.Nil(t, cycle)
	assert.Zero(t, f.cycles.created, "a disabled schedule records nothing")
	assert.Empty(t, f.corrections.countCalls)
}

func TestRetrainerPlanRecordsASkip(t *testing.T) {
	t.Parallel()

	f := newRetrainingFixture()
	f.corrections.trainable = 999

	cycle, err := f.service.Plan(t.Context(), scheduled())
	require.NoError(t, err)
	require.NotNil(t, cycle)
	assert.Equal(t, aitraining.RetrainingStatusSkipped, cycle.Status)
	assert.Equal(t, aitraining.RetrainingSkipNotEnoughExamples, cycle.SkipReason)
	assert.Equal(t, 999, cycle.NewExamples)
	assert.Equal(t, 1000, cycle.MinNewExamples)
	assert.Empty(t, f.operator.started)

	require.Len(t, f.corrections.countCalls, 1)
	count := f.corrections.countCalls[0]
	assert.Equal(t, testNow-365*day, count.CapturedFrom, "a first cycle counts the whole window")
	assert.Equal(t, testNow, count.CapturedTo)
	assert.Equal(t, 2000, count.PerOrganizationCap)
}

func TestRetrainerPlanStartsAnExport(t *testing.T) {
	t.Parallel()

	f := newRetrainingFixture()
	f.corrections.trainable = 1500
	lastWindowEnd := testNow - 40*day
	f.cycles.add(&aitraining.RetrainingCycle{
		Status:     aitraining.RetrainingStatusRejected,
		CreatedAt:  lastWindowEnd,
		CapturedTo: lastWindowEnd,
	})

	cycle, err := f.service.Plan(t.Context(), scheduled())
	require.NoError(t, err)
	require.NotNil(t, cycle)
	assert.Equal(t, aitraining.RetrainingStatusExporting, cycle.Status)
	assert.Equal(t, aitraining.RetrainingTriggerScheduled, cycle.Trigger)
	assert.Equal(t, aiprovider.StructuredOutputJSONSchema, cycle.StructuredOutputMode)
	assert.Equal(t, 85, cycle.MinAccuracyPercent)
	require.NotNil(t, cycle.ExportID)

	require.Len(t, f.corrections.countCalls, 1)
	assert.Equal(t, lastWindowEnd, f.corrections.countCalls[0].CapturedFrom,
		"only corrections after the last trained window are new")

	require.Len(t, f.operator.started, 1)
	started := f.operator.started[0]
	assert.Equal(t, testNow-365*day, started.CapturedFrom, "training reads the whole lookback")
	assert.Equal(t, testNow, started.CapturedTo)
	assert.Equal(t, 10, started.ValidationPercent)
	assert.Equal(t, aitraining.RetrainingRequestedByScheduler, started.RequestedBy)
	assert.Equal(t, "Retraining cycle "+cycle.ID.String(), started.Note)

	stored := f.cycles.find(cycle.ID)
	require.NotNil(t, stored)
	assert.Equal(t, *cycle.ExportID, *stored.ExportID)
}

func TestRetrainerPlanFailsTheCycleWhenTheExportCannotStart(t *testing.T) {
	t.Parallel()

	f := newRetrainingFixture()
	f.corrections.trainable = 5000
	f.operator.startErr = errStartRefused

	cycle, err := f.service.Plan(t.Context(), scheduled())
	require.ErrorIs(t, err, errStartRefused)
	require.NotNil(t, cycle)
	assert.Equal(t, aitraining.RetrainingStatusFailed, cycle.Status)
	assert.Equal(t, "The training export could not be started: temporal unavailable", cycle.FailureMessage)
	assert.Equal(t, aitraining.RetrainingStatusFailed, f.cycles.find(cycle.ID).Status,
		"the failed cycle frees the slot for the next one")
}

func driftingTotals(now int64) []aicorrection.WeekTotal {
	window := aicorrection.NewTrendWindow(now)
	provider := pulid.MustNew("aip_")
	totals := []aicorrection.WeekTotal{
		{ProviderID: provider, WeekStart: window.CheckedWeek, Scored: 100, Correct: 80},
	}
	for week := window.BaselineStart; week < window.CheckedWeek; week += 7 * day {
		totals = append(totals, aicorrection.WeekTotal{
			ProviderID: provider, WeekStart: week, Scored: 50, Correct: 45,
		})
	}
	return totals
}

func TestRetrainerPlanRetrainsEarlyOnDrift(t *testing.T) {
	t.Parallel()

	f := newRetrainingFixture()
	f.corrections.trainable = 1200
	f.corrections.weekly = driftingTotals(testNow)
	f.cycles.add(&aitraining.RetrainingCycle{
		Status:     aitraining.RetrainingStatusPassed,
		CreatedAt:  testNow - 3*day,
		CapturedTo: testNow - 3*day,
	})

	cycle, err := f.service.Plan(t.Context(), scheduled())
	require.NoError(t, err)
	assert.Equal(t, aitraining.RetrainingTriggerDrift, cycle.Trigger)
	assert.Equal(t, aitraining.RetrainingStatusExporting, cycle.Status)
	assert.Equal(t, 1, cycle.DriftingProviders)

	f2 := newRetrainingFixture()
	off := false
	f2.cfg.AIRetraining.RetrainOnDrift = &off
	f2.corrections.trainable = 1200
	f2.corrections.weekly = driftingTotals(testNow)
	f2.cycles.add(&aitraining.RetrainingCycle{
		Status:     aitraining.RetrainingStatusPassed,
		CreatedAt:  testNow - 3*day,
		CapturedTo: testNow - 3*day,
	})
	skipped, err := f2.service.Plan(t.Context(), scheduled())
	require.NoError(t, err)
	assert.Equal(t, aitraining.RetrainingSkipTooSoon, skipped.SkipReason)
	assert.Zero(t, f2.corrections.weeklyCalls, "drift is not read when it cannot trigger")
}

func TestRetrainerPlanSkipsWhileAnotherExportRuns(t *testing.T) {
	t.Parallel()

	f := newRetrainingFixture()
	f.corrections.trainable = 5000
	_, err := f.exports.Create(t.Context(), &aitraining.TrainingExport{Status: aitraining.ExportStatusRunning})
	require.NoError(t, err)

	cycle, err := f.service.Plan(t.Context(), scheduled())
	require.NoError(t, err)
	assert.Equal(t, aitraining.RetrainingSkipExportActive, cycle.SkipReason)
	assert.Empty(t, f.corrections.countCalls, "nothing is counted when the cycle cannot start")
}

func TestRetrainerManualPlan(t *testing.T) {
	t.Parallel()

	f := newRetrainingFixture()
	f.cfg.AIRetraining.Enabled = false

	cycle, err := f.service.Plan(t.Context(), &services.PlanAIRetrainingRequest{
		Manual:      true,
		RequestedBy: "  Jordan Lee ",
		Note:        "new customer layouts",
	})
	require.NoError(t, err, "an operator can retrain even with the schedule off")
	assert.Equal(t, aitraining.RetrainingTriggerManual, cycle.Trigger)
	assert.Equal(t, "Jordan Lee", cycle.RequestedBy)
	require.Len(t, f.operator.started, 1)
	assert.Equal(t, "Retraining cycle "+cycle.ID.String()+": new customer layouts",
		f.operator.started[0].Note)
	assert.Zero(t, f.corrections.weeklyCalls)

	created := f.cycles.created
	_, err = f.service.Plan(t.Context(), &services.PlanAIRetrainingRequest{Manual: true, RequestedBy: "ops"})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "already exporting")
	assert.Equal(t, created, f.cycles.created, "a refused manual request records nothing")
}

func exportingCycle(
	t *testing.T,
	f *retrainingFixture,
	export *aitraining.TrainingExport,
) *aitraining.RetrainingCycle {
	t.Helper()

	cycle := &aitraining.RetrainingCycle{
		Status:    aitraining.RetrainingStatusExporting,
		CreatedAt: f.now - 10,
	}
	if export != nil {
		created, err := f.exports.Create(t.Context(), export)
		require.NoError(t, err)
		cycle.ExportID = &created.ID
	}
	return f.cycles.add(cycle)
}

func TestRetrainerReconcile(t *testing.T) {
	t.Parallel()

	t.Run("completed export becomes ready", func(t *testing.T) {
		t.Parallel()
		f := newRetrainingFixture()
		cycle := exportingCycle(t, f, &aitraining.TrainingExport{
			Status: aitraining.ExportStatusCompleted, TrainExamples: 90, ValidationExamples: 10,
		})
		changed, err := f.service.Reconcile(t.Context())
		require.NoError(t, err)
		assert.Equal(t, 1, changed)
		assert.Equal(t, aitraining.RetrainingStatusReady, f.cycles.find(cycle.ID).Status)
	})

	t.Run("running export is left alone", func(t *testing.T) {
		t.Parallel()
		f := newRetrainingFixture()
		cycle := exportingCycle(t, f, &aitraining.TrainingExport{Status: aitraining.ExportStatusRunning})
		changed, err := f.service.Reconcile(t.Context())
		require.NoError(t, err)
		assert.Zero(t, changed)
		assert.Equal(t, aitraining.RetrainingStatusExporting, f.cycles.find(cycle.ID).Status)
	})

	t.Run("a removed export fails the cycle", func(t *testing.T) {
		t.Parallel()
		f := newRetrainingFixture()
		missing := pulid.MustNew("aitx_")
		cycle := f.cycles.add(&aitraining.RetrainingCycle{
			Status:   aitraining.RetrainingStatusExporting,
			ExportID: &missing,
		})
		_, err := f.service.Reconcile(t.Context())
		require.NoError(t, err)
		stored := f.cycles.find(cycle.ID)
		assert.Equal(t, aitraining.RetrainingStatusFailed, stored.Status)
		assert.Equal(t, "The training export no longer exists", stored.FailureMessage)
	})

	t.Run("an export that never started fails only after the grace period", func(t *testing.T) {
		t.Parallel()
		f := newRetrainingFixture()
		cycle := exportingCycle(t, f, nil)
		_, err := f.service.Reconcile(t.Context())
		require.NoError(t, err)
		assert.Equal(t, aitraining.RetrainingStatusExporting, f.cycles.find(cycle.ID).Status)

		f.now += int64(time.Hour / time.Second)
		_, err = f.service.Reconcile(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "The training export was never started", f.cycles.find(cycle.ID).FailureMessage)
	})
}

func readyCycle(f *retrainingFixture) *aitraining.RetrainingCycle {
	return f.cycles.add(&aitraining.RetrainingCycle{
		Status:             aitraining.RetrainingStatusReady,
		MinAccuracyPercent: 85,
	})
}

func TestRetrainerClaimAndLease(t *testing.T) {
	t.Parallel()

	f := newRetrainingFixture()
	claimed, err := f.service.ClaimNext(t.Context(), &services.ClaimAIRetrainingRequest{Trainer: "gpu-1"})
	require.NoError(t, err)
	assert.Nil(t, claimed, "nothing to train")

	_, err = f.service.ClaimNext(t.Context(), &services.ClaimAIRetrainingRequest{Trainer: "  "})
	require.Error(t, err)

	cycle := readyCycle(f)
	claimed, err = f.service.ClaimNext(t.Context(), &services.ClaimAIRetrainingRequest{Trainer: " gpu-1 "})
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, cycle.ID, claimed.ID)
	assert.Equal(t, "gpu-1", claimed.Trainer)
	assert.Equal(t, testNow+int64(2*time.Hour/time.Second), *claimed.LeaseExpiresAt)

	other, err := f.service.ClaimNext(t.Context(), &services.ClaimAIRetrainingRequest{Trainer: "gpu-2"})
	require.NoError(t, err)
	assert.Nil(t, other, "a held lease is not handed out twice")

	f.now += 3600
	extended, err := f.service.Heartbeat(t.Context(), &services.HeartbeatAIRetrainingRequest{
		CycleID: cycle.ID, Trainer: "gpu-1",
	})
	require.NoError(t, err)
	assert.Equal(t, f.now+int64(2*time.Hour/time.Second), *extended.LeaseExpiresAt)

	_, err = f.service.Heartbeat(t.Context(), &services.HeartbeatAIRetrainingRequest{
		CycleID: cycle.ID, Trainer: "gpu-2",
	})
	require.ErrorIs(t, err, aitraining.ErrRetrainingLeaseLost)

	f.now = *extended.LeaseExpiresAt
	taken, err := f.service.ClaimNext(t.Context(), &services.ClaimAIRetrainingRequest{Trainer: "gpu-2"})
	require.NoError(t, err)
	require.NotNil(t, taken, "an expired lease goes to the next trainer")
	assert.Equal(t, 2, taken.Attempts)

	_, err = f.service.Record(t.Context(), &services.RecordAIRetrainingRequest{
		CycleID: cycle.ID,
		Trainer: "gpu-1",
		Result:  &aitraining.RetrainingResult{Report: &aitraining.ScoreReport{}},
	})
	require.ErrorIs(t, err, aitraining.ErrRetrainingLeaseLost, "the old trainer cannot record a result")
}

func TestRetrainerClaimReconcilesAFinishedExportFirst(t *testing.T) {
	t.Parallel()

	f := newRetrainingFixture()
	cycle := exportingCycle(t, f, &aitraining.TrainingExport{
		Status: aitraining.ExportStatusCompleted, TrainExamples: 9, ValidationExamples: 1,
	})

	claimed, err := f.service.ClaimNext(t.Context(), &services.ClaimAIRetrainingRequest{Trainer: "gpu-1"})
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, cycle.ID, claimed.ID)
	assert.Equal(t, aitraining.RetrainingStatusTraining, claimed.Status)
}

func TestRetrainerRecordAndFail(t *testing.T) {
	t.Parallel()

	f := newRetrainingFixture()
	cycle := readyCycle(f)
	_, err := f.service.ClaimNext(t.Context(), &services.ClaimAIRetrainingRequest{Trainer: "gpu-1"})
	require.NoError(t, err)

	recorded, err := f.service.Record(t.Context(), &services.RecordAIRetrainingRequest{
		CycleID: cycle.ID,
		Trainer: "gpu-1",
		Result: &aitraining.RetrainingResult{
			TrainingConfig: "configs/qwen2.5-7b-instruct.yaml",
			ModelDirectory: "/data/runs/x/dpo/model",
			Report: &aitraining.ScoreReport{
				Examples: 100,
				Model:    aitraining.ScoreSide{Correct: 90, Scored: 100},
				Baseline: aitraining.ScoreSide{Correct: 88, Scored: 100},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, aitraining.RetrainingStatusPassed, recorded.Status)
	assert.Equal(t, "/data/runs/x/dpo/model", f.cycles.find(cycle.ID).ModelDirectory)

	second := readyCycle(f)
	_, err = f.service.ClaimNext(t.Context(), &services.ClaimAIRetrainingRequest{Trainer: "gpu-1"})
	require.NoError(t, err)
	failed, err := f.service.FailTraining(t.Context(), &services.FailAIRetrainingRequest{
		CycleID: second.ID, Trainer: "gpu-1", Message: "trenova-finetune run exited with status 1",
	})
	require.NoError(t, err)
	assert.Equal(t, aitraining.RetrainingStatusFailed, failed.Status)
	assert.Equal(t, "trenova-finetune run exited with status 1", failed.FailureMessage)
}

func TestRetrainerCancel(t *testing.T) {
	t.Parallel()

	f := newRetrainingFixture()
	cycle := exportingCycle(t, f, &aitraining.TrainingExport{Status: aitraining.ExportStatusRunning})

	canceled, err := f.service.Cancel(t.Context(), cycle.ID)
	require.NoError(t, err)
	assert.Equal(t, aitraining.RetrainingStatusCanceled, canceled.Status)
	assert.Equal(t, []pulid.ID{*cycle.ExportID}, f.operator.canceled, "its export is canceled too")

	_, err = f.service.Cancel(t.Context(), cycle.ID)
	require.ErrorIs(t, err, aitraining.ErrRetrainingClosed)

	finished := exportingCycle(t, f, &aitraining.TrainingExport{Status: aitraining.ExportStatusCompleted})
	_, err = f.service.Cancel(t.Context(), finished.ID)
	require.NoError(t, err, "an export that already finished does not stop the cancel")
}
