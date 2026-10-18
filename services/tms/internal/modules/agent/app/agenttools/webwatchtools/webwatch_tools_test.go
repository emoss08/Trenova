package webwatchtools

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentextensionservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const endsAt = int64(1_792_742_400)

type fakeWatcher struct {
	planErr error
	setErr  error
	planned []*agentextensionports.WebWatchRequest
	set     []*agentextensionports.WebWatchRequest
	stopped []*agentextensionports.StopWebWatchRequest
	watchID pulid.ID
}

func newFakeWatcher() *fakeWatcher {
	return &fakeWatcher{watchID: pulid.MustNew(agentextension.WebWatchIDPrefix)}
}

func (f *fakeWatcher) PlanWebWatch(
	_ context.Context,
	req *agentextensionports.WebWatchRequest,
) (*agentextensionports.WebWatchPlan, error) {
	f.planned = append(f.planned, req)
	if f.planErr != nil {
		return nil, f.planErr
	}

	return &agentextensionports.WebWatchPlan{
		Query:          req.Query,
		Purpose:        req.Purpose,
		Sites:          req.Sites,
		Cadence:        agentextension.WebWatchCadenceWeekly,
		Days:           28,
		EndsAt:         endsAt,
		Vendor:         "Exa",
		CostPerCheck:   agentextension.ExaMonitorRunCostUSD,
		EstimatedCost:  decimal.RequireFromString("0.06"),
		ActiveWatches:  1,
		WatchesAllowed: 10,
	}, nil
}

func (f *fakeWatcher) WatchWeb(
	_ context.Context,
	req *agentextensionports.WebWatchRequest,
) (*agentextensionports.WebWatchView, error) {
	f.set = append(f.set, req)
	if f.setErr != nil {
		return nil, f.setErr
	}

	return &agentextensionports.WebWatchView{
		ID:      f.watchID,
		Purpose: req.Purpose,
		Cadence: agentextension.WebWatchCadenceWeekly,
		Status:  agentextension.WebWatchStatusActive,
		EndsAt:  endsAt,
	}, nil
}

func (f *fakeWatcher) ListWebWatches(
	context.Context,
	*agentextensionports.ListWebWatchesRequest,
) (*agentextensionports.WebWatchListResponse, error) {
	return &agentextensionports.WebWatchListResponse{}, nil
}

func (f *fakeWatcher) StopWebWatch(
	_ context.Context,
	req *agentextensionports.StopWebWatchRequest,
) (*agentextensionports.WebWatchView, error) {
	f.stopped = append(f.stopped, req)

	return &agentextensionports.WebWatchView{
		ID:      req.WatchID,
		Purpose: "FMCSA finalizing the broker transparency rule",
		Status:  agentextension.WebWatchStatusStopped,
	}, nil
}

type fakeWaits struct {
	registered []*agentextensionports.RegisterWebWatchWaitRequest
	ended      []*agentextension.WebWatch
	err        error
}

func (f *fakeWaits) RegisterWebWatchWait(
	_ context.Context,
	req *agentextensionports.RegisterWebWatchWaitRequest,
) error {
	f.registered = append(f.registered, req)

	return f.err
}

func (f *fakeWaits) NotifyWebWatch(
	context.Context,
	pagination.TenantInfo,
	*agentextension.WebWatch,
	[]*agentextension.WebWatchFinding,
) error {
	return nil
}

func (f *fakeWaits) EndWebWatchWaits(
	_ context.Context,
	_ pagination.TenantInfo,
	watch *agentextension.WebWatch,
) {
	f.ended = append(f.ended, watch)
}

func watchArgs() map[string]any {
	return map[string]any{
		paramPurpose: "FMCSA finalizing the broker transparency rule",
		paramQuery:   "FMCSA final rule broker transparency 49 CFR 371.3",
		paramSites:   []any{"fmcsa.dot.gov", "federalregister.gov"},
		paramCadence: "Weekly",
		paramDays:    float64(28),
	}
}

func personParams(args map[string]any) agentports.ToolExecuteParams {
	params := agenttoolstest.DeskParams(args)
	params.ThreadID = pulid.MustNew("athr_")

	return params
}

func TestWatchingTheWebRunsOnItsOwnAndIsHeldAfterOutsideContent(t *testing.T) {
	t.Parallel()

	policy := NewWatchWebTool(newFakeWatcher(), &fakeWaits{}, nil).Policy()

	assert.Equal(t, agent.ToolKindAction, policy.Kind)
	assert.Equal(t, agent.TierAutoExecute, policy.DefaultTier)
	assert.Equal(t, permission.ResourceWebResearch, policy.Resource)
	assert.Equal(t, permission.OpRead, policy.Operation)
	assert.True(t, policy.CarriesTaint)
	assert.Equal(t, string(permission.KindWebWatch), policy.Artifact)
}

func TestTheWatchPreviewSaysWhatItSearchesHowOftenAndWhatItCosts(t *testing.T) {
	t.Parallel()

	watcher := newFakeWatcher()
	tool := NewWatchWebTool(watcher, &fakeWaits{}, nil).(*WatchWebTool)

	preview, err := tool.Preview(t.Context(), personParams(watchArgs()))
	require.NoError(t, err)

	assert.Empty(t, preview.Warnings)
	assert.Equal(t,
		"Would ask Exa to search 2 site(s) once a week until Oct 23, 2026 for FMCSA finalizing "+
			"the broker transparency rule, with the search \"FMCSA final rule broker "+
			"transparency 49 CFR 371.3\", and bring back only what it has not found before. "+
			"About $0.06 in all at Exa's published price ($0.015 a check).",
		preview.Summary)
	require.Len(t, watcher.planned, 1)
	assert.Equal(t, []string{"fmcsa.dot.gov", "federalregister.gov"}, watcher.planned[0].Sites)
	assert.Equal(t, 28, watcher.planned[0].Days)
	assert.Empty(t, watcher.set, "a preview never sets the watch")
}

func TestTheWatchPreviewWarnsWhenTheWatchWouldBeRefused(t *testing.T) {
	t.Parallel()

	watcher := newFakeWatcher()
	watcher.planErr = &agentextensionservice.ToolError{
		Message: "Parallel cannot watch the web on a schedule.",
	}
	tool := NewWatchWebTool(watcher, &fakeWaits{}, nil).(*WatchWebTool)

	preview, err := tool.Preview(t.Context(), personParams(watchArgs()))
	require.NoError(t, err)
	require.NotEmpty(t, preview.Warnings)
}

func TestAnUnknownCadenceIsRefusedWithTheOnesThatExist(t *testing.T) {
	t.Parallel()

	args := watchArgs()
	args[paramCadence] = "Hourly"
	tool := NewWatchWebTool(newFakeWatcher(), &fakeWaits{}, nil).(*WatchWebTool)

	err := tool.Validate(t.Context(), personParams(args))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Weekly")
}

func TestSettingAWatchParksTheConversationOnIt(t *testing.T) {
	t.Parallel()

	watcher := newFakeWatcher()
	waits := &fakeWaits{}
	tool := NewWatchWebTool(watcher, waits, nil).(*WatchWebTool)
	params := personParams(watchArgs())

	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, "set", result.Action)
	assert.Equal(t, watcher.watchID.String(), result.IDs["webWatchId"])
	assert.Equal(t, &agent.RecordRef{
		EntityType: string(permission.KindWebWatch),
		ID:         watcher.watchID.String(),
	}, result.Record)
	assert.Equal(t, "Watching once a week until Oct 23, 2026. What it finds comes back to "+
		"this work as it is found.", result.State)

	require.Len(t, waits.registered, 1)
	registered := waits.registered[0]
	assert.Equal(t, watcher.watchID, registered.Watch.ID)
	assert.Equal(t, params.ThreadID, registered.Owner.ThreadID)
	assert.Empty(t, watcher.stopped)
}

func TestAWatchWhoseWorkCannotBeParkedIsStoppedAtOnce(t *testing.T) {
	t.Parallel()

	watcher := newFakeWatcher()
	waits := &fakeWaits{err: errors.New("this conversation has ten open waits")}
	tool := NewWatchWebTool(watcher, waits, nil).(*WatchWebTool)

	_, err := tool.ExecuteWithResult(t.Context(), personParams(watchArgs()))
	require.Error(t, err)

	require.Len(t, watcher.stopped, 1, "what it found would reach no one, and bill for nothing")
	assert.Equal(t, watcher.watchID, watcher.stopped[0].WatchID)
}

func TestADelegatedTaskCannotSetAWatch(t *testing.T) {
	t.Parallel()

	watcher := newFakeWatcher()
	tool := NewWatchWebTool(watcher, &fakeWaits{}, nil).(*WatchWebTool)
	params := personParams(watchArgs())
	params.Delegated = true

	_, err := tool.ExecuteWithResult(t.Context(), params)
	require.ErrorIs(t, err, ErrDelegatedWatch)
	assert.Empty(t, watcher.set)
}

func TestStoppingAWatchClosesTheWaitOnIt(t *testing.T) {
	t.Parallel()

	watcher := newFakeWatcher()
	waits := &fakeWaits{}
	tool := NewStopWebWatchTool(watcher, waits).(*StopWebWatchTool)
	params := personParams(map[string]any{paramWatchID: watcher.watchID.String()})

	result, err := tool.ExecuteWithResult(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, "stopped", result.Action)
	assert.Equal(t, "The watch is Stopped.", result.State)
	require.Len(t, watcher.stopped, 1)
	assert.Equal(t, params.Actor.UserID, watcher.stopped[0].StoppedBy)
	assert.False(t, watcher.stopped[0].Administers, "an agent stops only what it may")
	require.Len(t, waits.ended, 1)
	assert.Equal(t, agentextension.WebWatchStatusStopped, waits.ended[0].Status)
}
