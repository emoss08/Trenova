package webwatchjobs

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

type scriptedRunner struct {
	checks map[pulid.ID]*agentextensionports.WebWatchCheck
	fail   map[pulid.ID]bool
}

func (r *scriptedRunner) CheckWebWatch(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*agentextensionports.WebWatchCheck, error) {
	if r.fail[id] {
		return nil, errors.New("vendor unavailable")
	}

	return r.checks[id], nil
}

func (r *scriptedRunner) MarkWebWatchDelivered(
	context.Context,
	pagination.TenantInfo,
	[]*extensionwebwatch.WebWatchFinding,
) error {
	return nil
}

type recordingWaits struct {
	agentextensionports.WebWatchWaits

	mu    sync.Mutex
	ended []pulid.ID
}

func (w *recordingWaits) EndWebWatchWaits(
	_ context.Context,
	_ pagination.TenantInfo,
	watch *extensionwebwatch.WebWatch,
) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ended = append(w.ended, watch.ID)
}

type sweepWatches struct {
	agentextensionports.WebWatchRepository

	mu        sync.Mutex
	due       []agentextensionports.WebWatchRef
	claimed   *agentextensionports.ClaimWebWatchChecksRequest
	scheduled map[pulid.ID]int64
	cleared   []pulid.ID
}

func (w *sweepWatches) ClaimDueChecks(
	_ context.Context,
	req *agentextensionports.ClaimWebWatchChecksRequest,
) ([]agentextensionports.WebWatchRef, error) {
	w.claimed = req

	return w.due, nil
}

func (w *sweepWatches) ScheduleCheck(_ context.Context, _ pagination.TenantInfo, id pulid.ID, at int64) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.scheduled[id] = at

	return nil
}

func (w *sweepWatches) ClearCheck(_ context.Context, _ pagination.TenantInfo, id pulid.ID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.cleared = append(w.cleared, id)

	return nil
}

// A sweep claims the due watches with their next check half an hour out,
// takes a watch found ended off the sweep, tells the work waiting on a
// stopped one, and brings a failed check forward rather than ending it.
func TestCheckDueWebWatchesSettlesEachWatch(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	active, stopped, failing := pulid.MustNew("aww_"), pulid.MustNew("aww_"), pulid.MustNew("aww_")
	runner := &scriptedRunner{
		checks: map[pulid.ID]*agentextensionports.WebWatchCheck{
			active: {Watch: &extensionwebwatch.WebWatch{ID: active, Status: extensionwebwatch.WebWatchStatusActive}},
			stopped: {
				Watch: &extensionwebwatch.WebWatch{ID: stopped, Status: extensionwebwatch.WebWatchStatusStopped},
				Ended: true,
			},
		},
		fail: map[pulid.ID]bool{failing: true},
	}
	waits := &recordingWaits{}
	watches := &sweepWatches{
		due: []agentextensionports.WebWatchRef{
			{ID: active, Tenant: tenant},
			{ID: stopped, Tenant: tenant},
			{ID: failing, Tenant: tenant},
		},
		scheduled: map[pulid.ID]int64{},
	}
	a := NewActivities(ActivitiesParams{Runner: runner, Waits: waits, Watches: watches, Logger: zap.NewNop()})

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(a)
	encoded, err := env.ExecuteActivity(a.CheckDueWebWatchesActivity)
	require.NoError(t, err)
	var result SweepResult
	require.NoError(t, encoded.Get(&result))

	assert.Equal(t, SweepResult{Checked: 3, Ended: 1, Failed: 1}, result)
	require.NotNil(t, watches.claimed)
	assert.Equal(t, int64(checkEvery.Seconds()), watches.claimed.Next-watches.claimed.Now)
	assert.Equal(t, []pulid.ID{stopped}, watches.cleared)
	assert.Equal(t, []pulid.ID{stopped}, waits.ended)
	require.Contains(t, watches.scheduled, failing)
	assert.NotContains(t, watches.scheduled, active, "a checked watch keeps the next check its claim set")
}
