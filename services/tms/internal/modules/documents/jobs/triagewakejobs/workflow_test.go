package triagewakejobs

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/triagewakejobs/triagewakecontract"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

type scriptedWorker struct {
	mu        sync.Mutex
	schedules []triagewakecontract.Schedule
	wakes     []triagewakecontract.WakeResult
	wokenAt   []int64
	reads     int
}

func (w *scriptedWorker) WakeSchedule(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*triagewakecontract.Schedule, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.reads++
	next := w.schedules[0]
	if len(w.schedules) > 1 {
		w.schedules = w.schedules[1:]
	}

	return &next, nil
}

func (w *scriptedWorker) WakeIfDue(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	now int64,
) (*triagewakecontract.WakeResult, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.wokenAt = append(w.wokenAt, now)
	next := w.wakes[0]
	w.wakes = w.wakes[1:]

	return &next, nil
}

func (w *scriptedWorker) ReconcileOverdueWakes(context.Context, int64) (int, error) {
	return 0, nil
}

func (w *scriptedWorker) RecomputeDueFor(
	context.Context,
	pagination.TenantInfo,
	[]inboundmessage.Classification,
) (int64, error) {
	return 0, nil
}

func run(t *testing.T, worker *scriptedWorker, during func(*testsuite.TestWorkflowEnvironment)) {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(NewActivities(ActivitiesParams{Worker: worker}))
	env.RegisterWorkflow(TriageWakeWorkflow)
	if during != nil {
		during(env)
	}

	env.ExecuteWorkflow(TriageWakeWorkflow, &Payload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		MessageID: pulid.MustNew("imsg_"),
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestTriageWakeWorkflow_WakesTheMessageWhenItsTimeComes(t *testing.T) {
	t.Parallel()

	var start time.Time
	worker := &scriptedWorker{wakes: []triagewakecontract.WakeResult{{Woken: true}}}
	run(t, worker, func(env *testsuite.TestWorkflowEnvironment) {
		start = env.Now()
		worker.schedules = []triagewakecontract.Schedule{{
			Open:   true,
			WakeAt: start.Add(2 * time.Hour).Unix(),
		}}
	})

	require.Len(t, worker.wokenAt, 1)
	assert.GreaterOrEqual(t, worker.wokenAt[0], start.Add(2*time.Hour).Unix(),
		"it sleeps to the snooze's end, never polling before it")
}

func TestTriageWakeWorkflow_EndsWhenThereIsNothingToWaitFor(t *testing.T) {
	t.Parallel()

	worker := &scriptedWorker{schedules: []triagewakecontract.Schedule{{}}}
	run(t, worker, nil)

	assert.Empty(t, worker.wokenAt)
	assert.Equal(t, 1, worker.reads)
}

func TestTriageWakeWorkflow_ReadsTheScheduleAgainWhenToldItMoved(t *testing.T) {
	t.Parallel()

	worker := &scriptedWorker{}
	run(t, worker, func(env *testsuite.TestWorkflowEnvironment) {
		now := env.Now()
		worker.schedules = []triagewakecontract.Schedule{
			{Open: true, WakeAt: now.Add(48 * time.Hour).Unix()},
			{},
		}
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(triagewakecontract.RescheduleSignal, nil)
		}, time.Hour)
	})

	assert.Empty(t, worker.wokenAt, "a person woke it before its time; the timer has nothing to do")
	assert.Equal(t, 2, worker.reads)
}

func TestTriageWakeWorkflow_SleepsAgainWhenTheSnoozeWasPutOff(t *testing.T) {
	t.Parallel()

	var start time.Time
	worker := &scriptedWorker{}
	run(t, worker, func(env *testsuite.TestWorkflowEnvironment) {
		start = env.Now()
		worker.schedules = []triagewakecontract.Schedule{{
			Open:   true,
			WakeAt: start.Add(time.Hour).Unix(),
		}}
		worker.wakes = []triagewakecontract.WakeResult{
			{Schedule: triagewakecontract.Schedule{
				Open:   true,
				WakeAt: start.Add(5 * time.Hour).Unix(),
			}},
			{Woken: true},
		}
	})

	require.Len(t, worker.wokenAt, 2, "an early check is not a wake-up; it is a new alarm")
	assert.GreaterOrEqual(t, worker.wokenAt[1], start.Add(5*time.Hour).Unix())
}
