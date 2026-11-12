package temporaljobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/internal/testutil/mocks/workqueueportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

const testDrainSignal = "test-drain"

type drainResult struct {
	Totals *DrainTotals `json:"totals"`
	Again  bool         `json:"again"`
}

// testDrainWorkflow runs the drain loop over a scripted list of pages.
func testDrainWorkflow(ctx workflow.Context, pages []DrainPage, iterations int) (*drainResult, error) {
	next := 0
	totals, again, err := RunTenantDrain(ctx, TenantDrainOptions{
		SignalName:       testDrainSignal,
		Debounce:         10 * time.Second,
		PagesPerRound:    3,
		IterationsPerRun: 5,
		Iterations:       iterations,
		Page: func(ctx workflow.Context) (*DrainPage, error) {
			if err := workflow.Sleep(ctx, time.Second); err != nil {
				return nil, err
			}
			if next >= len(pages) {
				return &DrainPage{}, nil
			}
			page := pages[next]
			next++

			return &page, nil
		},
	})

	return &drainResult{Totals: totals, Again: again}, err
}

func runDrain(t *testing.T, pages []DrainPage, iterations int, during func(*testsuite.TestWorkflowEnvironment)) *drainResult {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(testDrainWorkflow)
	if during != nil {
		during(env)
	}
	env.ExecuteWorkflow(testDrainWorkflow, pages, iterations)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result drainResult
	require.NoError(t, env.GetWorkflowResult(&result))

	return &result
}

// A burst is one round; a page that is not full ends it, and with no request
// meanwhile the drain ends.
func TestRunTenantDrainEndsWhenNothingIsLeft(t *testing.T) {
	t.Parallel()

	result := runDrain(t, []DrainPage{{Done: 4}}, 0, nil)

	assert.Equal(t, 4, result.Totals.Done)
	assert.False(t, result.Again)
}

// Full pages keep the round going; past the round's bound the next round
// starts at once, without waiting out another debounce.
func TestRunTenantDrainWorksABacklogPastTheRound(t *testing.T) {
	t.Parallel()

	full := DrainPage{Done: 10, Full: true}
	result := runDrain(t, []DrainPage{full, full, full, full, {Done: 2, Failed: 1}}, 0, nil)

	assert.Equal(t, 42, result.Totals.Done)
	assert.Equal(t, 1, result.Totals.Failed)
	assert.False(t, result.Again)
}

// A request that arrives during a round brings another round.
func TestRunTenantDrainRunsAgainForARequestDuringTheRound(t *testing.T) {
	t.Parallel()

	result := runDrain(t, []DrainPage{{Done: 1}, {Done: 1}}, 0, func(env *testsuite.TestWorkflowEnvironment) {
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(testDrainSignal, nil)
		}, 10*time.Second+500*time.Millisecond)
	})

	assert.Equal(t, 2, result.Totals.Done)
}

// A drain that has run its rounds asks to continue as new.
func TestRunTenantDrainContinuesAsNewAfterItsRounds(t *testing.T) {
	t.Parallel()

	result := runDrain(t, nil, 5, nil)

	assert.True(t, result.Again)
}

// DrainDue settles the subjects that were worked and pushes the failed ones
// back in one call, each with the due time it claimed.
func TestDrainDueSettlesEachSubject(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ok, bad := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	queue := workqueueportsmock.NewMockQueue(t)
	queue.EXPECT().ListDue(mock.Anything, mock.MatchedBy(func(req *workqueueports.ListDueRequest) bool {
		return req.Kind == "test" && req.Limit == 2 && req.TenantInfo == tenant
	})).Return([]workqueueports.Due{{SubjectID: ok, DueAt: 100}, {SubjectID: bad, DueAt: 101}}, nil).Once()
	queue.EXPECT().SettleMany(mock.Anything, mock.MatchedBy(func(req *workqueueports.SettleManyRequest) bool {
		return len(req.Done) == 1 && req.Done[0].SubjectID == ok && req.Done[0].DueAt == 100 &&
			len(req.Retry) == 1 && req.Retry[0].SubjectID == bad && req.Retry[0].DueAt == 101 &&
			req.RetryAt > 0
	})).Return(nil).Once()

	page, err := DrainDue(t.Context(), &DrainDueParams{
		Queue:      queue,
		Kind:       "test",
		Tenant:     tenant,
		PageSize:   2,
		RetryAfter: time.Minute,
		Handle: func(_ context.Context, subjects []pulid.ID, _ map[pulid.ID]string) map[pulid.ID]error {
			assert.Equal(t, []pulid.ID{ok, bad}, subjects)
			return map[pulid.ID]error{bad: errors.New("refused")}
		},
	})

	require.NoError(t, err)
	assert.Equal(t, &DrainPage{Done: 1, Failed: 1, Full: true}, page)
}

// A deferred subject is due again like a failed one but counts as blocked,
// not failed, so the drain comes back for it.
func TestDrainDueDefersWithoutFailing(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	waiting := pulid.MustNew("shp_")
	queue := workqueueportsmock.NewMockQueue(t)
	queue.EXPECT().ListDue(mock.Anything, mock.Anything).
		Return([]workqueueports.Due{{SubjectID: waiting, DueAt: 7, Detail: "kind"}}, nil).Once()
	queue.EXPECT().SettleMany(mock.Anything, mock.MatchedBy(func(req *workqueueports.SettleManyRequest) bool {
		return len(req.Done) == 0 && len(req.Retry) == 1 && req.Retry[0].SubjectID == waiting
	})).Return(nil).Once()

	page, err := DrainDue(t.Context(), &DrainDueParams{
		Queue:      queue,
		Kind:       "test",
		Tenant:     tenant,
		PageSize:   5,
		RetryAfter: time.Second,
		Handle: func(_ context.Context, subjects []pulid.ID, details map[pulid.ID]string) map[pulid.ID]error {
			assert.Equal(t, "kind", details[waiting], "each subject's detail reaches the handler")
			return map[pulid.ID]error{subjects[0]: ErrDeferred}
		},
	})

	require.NoError(t, err)
	assert.Equal(t, &DrainPage{Deferred: 1, Blocked: true}, page)
}

// A blocked round keeps the drain alive: it looks again after the debounce
// until the subjects go through, then ends.
func TestRunTenantDrainComesBackWhileBlocked(t *testing.T) {
	t.Parallel()

	result := runDrain(t, []DrainPage{{Deferred: 2, Blocked: true}, {Done: 2}}, 0, nil)

	assert.Equal(t, 2, result.Totals.Done)
	assert.False(t, result.Again)
}
