package agentwaitjobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentwaitjobs/agentwaitcontract"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// scriptedWorker answers the workflow from a script and records how the wait
// ended.
type scriptedWorker struct {
	mu       sync.Mutex
	schedule agentwaitcontract.Schedule
	checks   []agentwaitcontract.Check
	checked  []int64
	finished []agentwaitcontract.FinishInput
	// busy is how many pick-ups find the conversation busy before one lands;
	// broken fails every pick-up.
	busy     int
	broken   error
	attempts []time.Time
	now      func() time.Time
}

func (w *scriptedWorker) Schedule(context.Context, pagination.TenantInfo, pulid.ID) (*agentwaitcontract.Schedule, error) {
	return &w.schedule, nil
}

func (w *scriptedWorker) Check(_ context.Context, _ pagination.TenantInfo, _ pulid.ID, now int64) (*agentwaitcontract.Check, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.checked = append(w.checked, now)
	next := w.checks[0]
	w.checks = w.checks[1:]

	return &next, nil
}

func (w *scriptedWorker) ReconcileOverdue(context.Context) (int, error) { return 0, nil }

func (w *scriptedWorker) Finish(_ context.Context, in *agentwaitcontract.FinishInput) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.now != nil {
		w.attempts = append(w.attempts, w.now())
	}
	if w.broken != nil {
		return w.broken
	}
	if w.busy > 0 {
		w.busy--
		return agentwaitcontract.ErrConversationBusy
	}
	w.finished = append(w.finished, *in)

	return nil
}

func run(t *testing.T, worker *scriptedWorker, during func(*testsuite.TestWorkflowEnvironment)) {
	t.Helper()

	require.NoError(t, execute(t, worker, during))
}

func execute(
	t *testing.T,
	worker *scriptedWorker,
	during func(*testsuite.TestWorkflowEnvironment),
) error {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(NewActivities(ActivitiesParams{Worker: worker}))
	env.RegisterWorkflow(AgentWaitWorkflow)
	if during != nil {
		during(env)
	}

	env.ExecuteWorkflow(AgentWaitWorkflow, &agentwaitcontract.Payload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		WaitID: pulid.MustNew(agentwait.IDPrefix),
	})

	require.True(t, env.IsWorkflowCompleted())

	return env.GetWorkflowError()
}

func clock(env *testsuite.TestWorkflowEnvironment, after time.Duration) int64 {
	return env.Now().Add(after).Unix()
}

func TestAgentWaitWorkflow_AnEventEndsTheWait(t *testing.T) {
	t.Parallel()

	worker := &scriptedWorker{}
	run(t, worker, func(env *testsuite.TestWorkflowEnvironment) {
		worker.schedule = agentwaitcontract.Schedule{Open: true, ExpiresAt: clock(env, 24*time.Hour)}
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(agentwaitcontract.MetSignal, agentwaitcontract.Met{Detail: "Arrived at stop stp_1."})
		}, 2*time.Hour)
	})

	require.Len(t, worker.finished, 1)
	assert.Equal(t, agentwait.StatusMet, worker.finished[0].Status)
	assert.Equal(t, "Arrived at stop stp_1.", worker.finished[0].Outcome)
	assert.Empty(t, worker.checked, "an event-driven wait is never read on a timer before it ends")
}

func TestAgentWaitWorkflow_AMovedAppointmentPutsTheWaitOff(t *testing.T) {
	t.Parallel()

	worker := &scriptedWorker{}
	run(t, worker, func(env *testsuite.TestWorkflowEnvironment) {
		due, later := clock(env, time.Hour), clock(env, 3*time.Hour)
		worker.schedule = agentwaitcontract.Schedule{Open: true, DueAt: &due, ExpiresAt: clock(env, 24*time.Hour)}
		worker.checks = []agentwaitcontract.Check{
			{DueAt: &later},
			{Met: true, Detail: "The appointment at Kroger DC is at 16:00."},
		}
	})

	require.Len(t, worker.checked, 2, "read again when due, and again when due after the move")
	assert.Greater(t, worker.checked[1], worker.checked[0]+int64(time.Hour.Seconds()))
	require.Len(t, worker.finished, 1)
	assert.Equal(t, agentwait.StatusMet, worker.finished[0].Status)
	assert.Equal(t, "The appointment at Kroger DC is at 16:00.", worker.finished[0].Outcome)
}

func TestAgentWaitWorkflow_AWaitThatRunsOutIsPickedUpAnyway(t *testing.T) {
	t.Parallel()

	worker := &scriptedWorker{}
	run(t, worker, func(env *testsuite.TestWorkflowEnvironment) {
		worker.schedule = agentwaitcontract.Schedule{Open: true, ExpiresAt: clock(env, 6*time.Hour)}
	})

	require.Len(t, worker.finished, 1)
	assert.Equal(t, agentwait.StatusTimedOut, worker.finished[0].Status)
}

func TestAgentWaitWorkflow_ABusyConversationIsPickedUpOnceItIsFree(t *testing.T) {
	t.Parallel()

	worker := &scriptedWorker{busy: 3}
	run(t, worker, func(env *testsuite.TestWorkflowEnvironment) {
		worker.now = env.Now
		worker.schedule = agentwaitcontract.Schedule{Open: true, ExpiresAt: clock(env, time.Hour)}
	})

	require.Len(t, worker.finished, 1)
	assert.Equal(t, agentwait.StatusTimedOut, worker.finished[0].Status)
	require.Len(t, worker.attempts, 4, "each busy answer is one attempt, never retried by the policy")
	assert.Equal(t, busyWaitFirst, worker.attempts[1].Sub(worker.attempts[0]))
	assert.Greater(t, worker.attempts[3].Sub(worker.attempts[2]), worker.attempts[2].Sub(worker.attempts[1]),
		"the wait for a busy conversation backs off")
}

func TestAgentWaitWorkflow_ABrokenPickUpStopsAfterItsAttempts(t *testing.T) {
	t.Parallel()

	worker := &scriptedWorker{broken: errors.New("connection reset by peer")}
	err := execute(t, worker, func(env *testsuite.TestWorkflowEnvironment) {
		worker.now = env.Now
		worker.schedule = agentwaitcontract.Schedule{Open: true, ExpiresAt: clock(env, time.Hour)}
	})

	require.Error(t, err)
	assert.Empty(t, worker.finished)
	assert.Len(t, worker.attempts, int(finishOptions.RetryPolicy.MaximumAttempts),
		"a failure that is not a busy conversation is not retried for a day")
}

func TestAgentWaitWorkflow_AClosedWaitDoesNothing(t *testing.T) {
	t.Parallel()

	worker := &scriptedWorker{schedule: agentwaitcontract.Schedule{Open: false}}
	run(t, worker, nil)

	assert.Empty(t, worker.finished)
}

func TestAgentWaitWorkflow_AWaitClosedMeanwhileEndsQuietly(t *testing.T) {
	t.Parallel()

	worker := &scriptedWorker{}
	run(t, worker, func(env *testsuite.TestWorkflowEnvironment) {
		due := clock(env, time.Hour)
		worker.schedule = agentwaitcontract.Schedule{Open: true, DueAt: &due, ExpiresAt: clock(env, 24*time.Hour)}
		worker.checks = []agentwaitcontract.Check{{Closed: true}}
	})

	assert.Empty(t, worker.finished)
}

func TestAgentWaitWorkflow_AMovedDueTimeIsReadAgain(t *testing.T) {
	t.Parallel()

	worker := &scriptedWorker{}
	run(t, worker, func(env *testsuite.TestWorkflowEnvironment) {
		worker.schedule = agentwaitcontract.Schedule{Open: true, ExpiresAt: clock(env, 24*time.Hour)}
		env.RegisterDelayedCallback(func() {
			soon := clock(env, 30*time.Minute)
			worker.mu.Lock()
			worker.schedule = agentwaitcontract.Schedule{Open: true, DueAt: &soon, ExpiresAt: clock(env, 24*time.Hour)}
			worker.checks = []agentwaitcontract.Check{{Met: true, Detail: "Drive time left: about 1h 59m."}}
			worker.mu.Unlock()
			env.SignalWorkflow(agentwaitcontract.RescheduleSignal, nil)
		}, time.Hour)
	})

	require.Len(t, worker.checked, 1, "the new due time is slept to and read")
	require.Len(t, worker.finished, 1)
	assert.Equal(t, agentwait.StatusMet, worker.finished[0].Status)
	assert.Equal(t, "Drive time left: about 1h 59m.", worker.finished[0].Outcome)
}

func settling(env *testsuite.TestWorkflowEnvironment) agentwaitcontract.Schedule {
	return agentwaitcontract.Schedule{
		Open:             true,
		ExpiresAt:        clock(env, 24*time.Hour),
		SettleSeconds:    30,
		SettleMaxSeconds: 120,
	}
}

func TestAgentWaitWorkflow_ARecordEditedInABurstIsPickedUpOnceAfterTheLastEdit(t *testing.T) {
	t.Parallel()

	worker := &scriptedWorker{}
	var firstAt, lastAt time.Time
	run(t, worker, func(env *testsuite.TestWorkflowEnvironment) {
		worker.now = env.Now
		worker.schedule = settling(env)
		edit := func(after time.Duration, detail string, first bool) {
			env.RegisterDelayedCallback(func() {
				if first {
					firstAt = env.Now()
				}
				lastAt = env.Now()
				env.SignalWorkflow(agentwaitcontract.MetSignal, agentwaitcontract.Met{
					Key: "shp_1", Detail: detail, At: env.Now().Unix(),
				})
			}, after)
		}
		edit(time.Hour, "S-1042 was changed by someone else: status.", true)
		edit(time.Hour+10*time.Second, "S-1042 was changed by someone else: documents.", false)
		edit(time.Hour+25*time.Second, "S-1042 was changed by someone else: documents, pod.", false)
	})

	require.Len(t, worker.finished, 1, "three edits pick the work up once")
	assert.Equal(t, agentwait.StatusMet, worker.finished[0].Status)
	assert.Equal(t, "S-1042 was changed by someone else: documents, pod.",
		worker.finished[0].Outcome, "the latest notice per record is kept")
	assert.Equal(t, firstAt.Unix(), worker.finished[0].FirstMetAt)
	require.Len(t, worker.attempts, 1)
	assert.Equal(t, 30*time.Second, worker.attempts[0].Sub(lastAt),
		"picked up once the record has been quiet for the settle period")
}

func TestAgentWaitWorkflow_ARecordEditedWithoutPauseIsPickedUpAtTheLimit(t *testing.T) {
	t.Parallel()

	worker := &scriptedWorker{}
	var firstAt time.Time
	run(t, worker, func(env *testsuite.TestWorkflowEnvironment) {
		worker.now = env.Now
		worker.schedule = settling(env)
		for i := range 20 {
			env.RegisterDelayedCallback(func() {
				if i == 0 {
					firstAt = env.Now()
				}
				env.SignalWorkflow(agentwaitcontract.MetSignal, agentwaitcontract.Met{
					Key: "shp_1", Detail: "changed", At: env.Now().Unix(),
				})
			}, time.Hour+time.Duration(i)*10*time.Second)
		}
	})

	require.Len(t, worker.finished, 1)
	require.Len(t, worker.attempts, 1)
	assert.Equal(t, 120*time.Second, worker.attempts[0].Sub(firstAt),
		"a record never quiet still picks the work up at the settle limit")
}

func TestAgentWaitWorkflow_AKindThatDoesNotSettleIsPickedUpAtOnce(t *testing.T) {
	t.Parallel()

	worker := &scriptedWorker{}
	var metAt time.Time
	run(t, worker, func(env *testsuite.TestWorkflowEnvironment) {
		worker.now = env.Now
		worker.schedule = agentwaitcontract.Schedule{Open: true, ExpiresAt: clock(env, 24*time.Hour)}
		env.RegisterDelayedCallback(func() {
			metAt = env.Now()
			env.SignalWorkflow(agentwaitcontract.MetSignal, agentwaitcontract.Met{Detail: "Arrived."})
		}, time.Hour)
	})

	require.Len(t, worker.attempts, 1)
	assert.Equal(t, metAt, worker.attempts[0])
	assert.Zero(t, worker.finished[0].FirstMetAt)
}
