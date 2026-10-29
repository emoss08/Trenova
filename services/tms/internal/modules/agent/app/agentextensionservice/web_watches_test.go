package agentextensionservice

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/webwatchjobs/webwatchcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const (
	testWebBase       = "https://tms.example.com"
	testWatchSecret   = "whsec_test"
	testSignatureName = "X-Test-Signature"
)

type fakeWatcher struct {
	*fakeWeb

	mu        sync.Mutex
	created   []*agentextensionports.WebWatchCreateRequest
	deleted   []string
	runs      []agentextensionports.WebWatchRun
	disabled  bool
	createErr error
	deleteErr error
	after     []int64
}

func (f *fakeWatcher) CreateWatch(
	_ context.Context,
	req *agentextensionports.WebWatchCreateRequest,
) (*agentextensionports.WebWatchCreated, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.createErr != nil {
		return nil, f.createErr
	}
	f.created = append(f.created, req)

	return &agentextensionports.WebWatchCreated{VendorID: "mon_1", Secret: testWatchSecret}, nil
}

func (f *fakeWatcher) DeleteWatch(_ context.Context, _ map[string]string, vendorID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, vendorID)

	return nil
}

func (f *fakeWatcher) WatchState(
	context.Context,
	map[string]string,
	string,
) (*agentextensionports.WebWatchVendorState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return &agentextensionports.WebWatchVendorState{Disabled: f.disabled}, nil
}

func (f *fakeWatcher) WatchRuns(
	_ context.Context,
	req *agentextensionports.WebWatchRunsRequest,
) ([]agentextensionports.WebWatchRun, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.after = append(f.after, req.After)

	return slices.Clone(f.runs), nil
}

func (f *fakeWatcher) VerifyDelivery(
	delivery *agentextensionports.WebWatchDelivery,
	secret string,
) error {
	if delivery.Header.Get(testSignatureName) != secret {
		return errors.New("signature does not match")
	}

	return nil
}

type memoryWatches struct {
	mu      sync.Mutex
	watches map[pulid.ID]*extensionwebwatch.WebWatch
}

func (m *memoryWatches) Create(
	_ context.Context,
	watch *extensionwebwatch.WebWatch,
) (*extensionwebwatch.WebWatch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	watch.CreatedAt = testNow
	stored := *watch
	m.watches[watch.ID] = &stored

	return watch, nil
}

func (m *memoryWatches) Get(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*extensionwebwatch.WebWatch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	watch, ok := m.watches[id]
	if !ok {
		return nil, errortypes.NewNotFoundError("Web watch not found")
	}
	copied := *watch

	return &copied, nil
}

func (m *memoryWatches) Update(
	_ context.Context,
	watch *extensionwebwatch.WebWatch,
) (*extensionwebwatch.WebWatch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	watch.Version++
	stored := *watch
	m.watches[watch.ID] = &stored

	return watch, nil
}

func (m *memoryWatches) List(
	_ context.Context,
	_ *agentextensionports.ListWebWatchesRequest,
) ([]*extensionwebwatch.WebWatch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]*extensionwebwatch.WebWatch, 0, len(m.watches))
	for _, watch := range m.watches {
		copied := *watch
		out = append(out, &copied)
	}

	return out, nil
}

func (m *memoryWatches) CountActive(context.Context, pagination.TenantInfo) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, watch := range m.watches {
		if watch.Active() {
			count++
		}
	}

	return count, nil
}

func (m *memoryWatches) only(t *testing.T) *extensionwebwatch.WebWatch {
	t.Helper()

	m.mu.Lock()
	defer m.mu.Unlock()

	require.Len(t, m.watches, 1)
	for _, watch := range m.watches {
		copied := *watch
		return &copied
	}

	return nil
}

type memoryFindings struct {
	mu       sync.Mutex
	findings []*extensionwebwatch.WebWatchFinding
}

func (m *memoryFindings) Create(
	_ context.Context,
	finding *extensionwebwatch.WebWatchFinding,
) (*extensionwebwatch.WebWatchFinding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, known := range m.findings {
		if known.WatchID == finding.WatchID && known.VendorRunID == finding.VendorRunID {
			return nil, agentextensionports.ErrWebWatchFindingExists
		}
	}
	m.findings = append(m.findings, finding)

	return finding, nil
}

func (m *memoryFindings) ListByWatch(
	_ context.Context,
	_ pagination.TenantInfo,
	watchID pulid.ID,
	_ int,
) ([]*extensionwebwatch.WebWatchFinding, error) {
	return m.matching(
		func(f *extensionwebwatch.WebWatchFinding) bool { return f.WatchID == watchID },
	), nil
}

func (m *memoryFindings) ListUndelivered(
	_ context.Context,
	_ pagination.TenantInfo,
	watchID pulid.ID,
) ([]*extensionwebwatch.WebWatchFinding, error) {
	return m.matching(func(f *extensionwebwatch.WebWatchFinding) bool {
		return f.WatchID == watchID && f.DeliveredAt == nil
	}), nil
}

func (m *memoryFindings) MarkDelivered(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
	at int64,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, finding := range m.findings {
		if slices.Contains(ids, finding.ID) && finding.DeliveredAt == nil {
			delivered := at
			finding.DeliveredAt = &delivered
		}
	}

	return nil
}

func (m *memoryFindings) matching(
	keep func(*extensionwebwatch.WebWatchFinding) bool,
) []*extensionwebwatch.WebWatchFinding {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]*extensionwebwatch.WebWatchFinding, 0, len(m.findings))
	for _, finding := range m.findings {
		if keep(finding) {
			out = append(out, finding)
		}
	}

	return out
}

type watchWorkflows struct {
	mu      sync.Mutex
	started []string
	signals []string
}

func (w *watchWorkflows) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	_ ...any,
) (client.WorkflowRun, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.started = append(w.started, options.ID)

	return nil, nil //nolint:nilnil // the run handle is never read
}

func (w *watchWorkflows) CancelWorkflow(context.Context, string, string) error { return nil }

func (w *watchWorkflows) SignalWorkflow(
	_ context.Context,
	workflowID, _, signalName string,
	_ any,
) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.signals = append(w.signals, workflowID+"/"+signalName)

	return nil
}

func (w *watchWorkflows) Enabled() bool { return true }

type watchFixture struct {
	*fixture

	watcher   *fakeWatcher
	watches   *memoryWatches
	findings  *memoryFindings
	workflows *watchWorkflows
}

func newWatchFixture(t *testing.T, webBase string) *watchFixture {
	t.Helper()

	w := newBareWatchFixture(t, webBase)
	w.enable(t, map[string]string{"apiKey": testAPIKey})

	return w
}

// newBareWatchFixture is a watch fixture with no extension turned on.
func newBareWatchFixture(t *testing.T, webBase string) *watchFixture {
	t.Helper()

	f := newFixture(t)
	w := &watchFixture{
		fixture:   f,
		watcher:   &fakeWatcher{fakeWeb: f.web},
		watches:   &memoryWatches{watches: make(map[pulid.ID]*extensionwebwatch.WebWatch)},
		findings:  &memoryFindings{},
		workflows: &watchWorkflows{},
	}
	f.svc = New(Params{
		Logger:       zap.NewNop(),
		Repo:         f.repo,
		Usage:        f.usage,
		Encryption:   testEncryptionService(),
		AuditService: f.audit,
		Connectors: []agentextensionports.ExtensionConnector{
			w.watcher,
			f.parallel,
			f.browser,
		},
		Sites:     f.sites,
		Runs:      f.runs,
		Watches:   w.watches,
		Findings:  w.findings,
		Workflows: w.workflows,
		Config:    &config.Config{App: config.AppConfig{WebBaseURL: webBase}},
	})
	f.svc.now = func() int64 { return testNow }

	return w
}

func (w *watchFixture) request() *agentextensionports.WebWatchRequest {
	return &agentextensionports.WebWatchRequest{
		TenantInfo: w.tenant,
		Query:      "FMCSA final rule broker transparency 49 CFR 371.3",
		Purpose:    "FMCSA finalizing the broker transparency rule",
		Sites:      []string{"FMCSA.dot.gov", "federalregister.gov"},
		Cadence:    extensionwebwatch.WebWatchCadenceWeekly,
		Owner: agentextensionports.PortalTaskOwner{
			Actor: &permissionports.RequestActor{
				PrincipalType:  permissionports.PrincipalTypeUser,
				PrincipalID:    w.tenant.UserID,
				UserID:         w.tenant.UserID,
				OrganizationID: w.tenant.OrgID,
				BusinessUnitID: w.tenant.BuID,
			},
			ThreadID: pulid.MustNew("athr_"),
		},
	}
}

func (w *watchFixture) set(t *testing.T) *agentextensionports.WebWatchView {
	t.Helper()

	view, err := w.svc.WatchWeb(t.Context(), w.request())
	require.NoError(t, err)

	return view
}

func completedRun(id string, createdAt int64, urls ...string) agentextensionports.WebWatchRun {
	results := make([]extensionwebwatch.WebWatchResult, 0, len(urls))
	for _, target := range urls {
		results = append(
			results,
			extensionwebwatch.WebWatchResult{Title: "Final rule", URL: target},
		)
	}

	return agentextensionports.WebWatchRun{
		ID:        id,
		Phase:     agentextensionports.WebWatchRunCompleted,
		Summary:   "FMCSA published the final rule.",
		Results:   results,
		CreatedAt: createdAt,
		SettledAt: createdAt + 30,
		CostUSD:   extensionwebwatch.ExaMonitorRunCostUSD,
	}
}

func TestPlanningAWatchSaysWhatItWillCostWithoutReachingTheVendor(t *testing.T) {
	t.Parallel()

	w := newWatchFixture(t, testWebBase)
	req := w.request()
	req.Days = 28

	plan, err := w.svc.PlanWebWatch(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, []string{"fmcsa.dot.gov", "federalregister.gov"}, plan.Sites)
	assert.Equal(t, extensionwebwatch.WebWatchCadenceWeekly, plan.Cadence)
	assert.Equal(t, 28, plan.Days)
	assert.Equal(t, testNow+28*24*60*60, plan.EndsAt)
	assert.Equal(t, "Exa", plan.Vendor)
	assert.True(t, decimal.RequireFromString("0.06").Equal(plan.EstimatedCost),
		"four weekly checks at $0.015")
	assert.Equal(t, agentextension.DefaultMaxActiveWebWatches, plan.WatchesAllowed)
	assert.Empty(t, w.watcher.created)
	assert.Empty(t, w.watches.watches)
}

func TestSettingAWatchRecordsItThenFollowsIt(t *testing.T) {
	t.Parallel()

	w := newWatchFixture(t, testWebBase)
	view := w.set(t)

	assert.Equal(t, extensionwebwatch.WebWatchStatusActive, view.Status)
	require.Len(t, w.watcher.created, 1)
	created := w.watcher.created[0]
	assert.Equal(t, "Trenova: FMCSA finalizing the broker transparency rule", created.Name)
	assert.Equal(t, []string{"fmcsa.dot.gov", "federalregister.gov"}, created.IncludeDomains)
	assert.Empty(t, created.ExcludeDomains, "a watch that names its sites excludes none")
	assert.Equal(t, testWebBase+"/api/v1"+extensionwebwatch.WebWatchWebhookPath(
		w.tenant.OrgID, w.tenant.BuID, view.ID), created.WebhookURL)
	assert.Equal(t, view.ID.String(), created.Metadata["trenovaWebWatch"])

	stored := w.watches.only(t)
	assert.Equal(t, "mon_1", stored.VendorMonitorID)
	assert.Equal(t, w.tenant.UserID, stored.RequestedByID)
	assert.True(t, stored.HasSecret())
	assert.NotContains(t, stored.Secrets[extensionwebwatch.WebWatchSecretField], testWatchSecret,
		"the webhook secret is kept encrypted")
	assert.Equal(t, []string{webwatchcontract.WorkflowIDFor(view.ID)}, w.workflows.started)
}

func TestAWatchNeedsTrenovaAtAPublicAddress(t *testing.T) {
	t.Parallel()

	for _, base := range []string{
		"", "http://tms.example.com", "https://localhost:5173", "https://tms.local",
		"https://10.0.0.4", "https://127.0.0.1",
	} {
		w := newWatchFixture(t, base)
		_, err := w.svc.WatchWeb(t.Context(), w.request())

		require.Error(t, err, base)
		assert.Contains(t, err.Error(), "public https address", base)
		assert.Empty(t, w.watcher.created, base)
	}
}

func TestAWatchIsRefusedPastTheOrganizationsLimit(t *testing.T) {
	t.Parallel()

	w := newWatchFixture(t, testWebBase)
	w.enable(t, map[string]string{
		"apiKey":                                 testAPIKey,
		agentextension.ConfigKeyMaxActiveWatches: "1",
	})
	w.set(t)

	_, err := w.svc.WatchWeb(t.Context(), w.request())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "web_stop_watch")
	assert.Len(t, w.watcher.created, 1)
}

func TestAWatchTheVendorRefusedEndsFailedAtOnce(t *testing.T) {
	t.Parallel()

	w := newWatchFixture(t, testWebBase)
	w.watcher.createErr = errors.New("exa: 402 insufficient credits")

	_, err := w.svc.WatchWeb(t.Context(), w.request())
	require.Error(t, err)

	stored := w.watches.only(t)
	assert.Equal(t, extensionwebwatch.WebWatchStatusFailed, stored.Status)
	assert.NotEmpty(t, stored.Failure)
	assert.Empty(t, w.workflows.started)
}

func TestAVendorThatCannotWatchIsRefused(t *testing.T) {
	t.Parallel()

	w := newBareWatchFixture(t, testWebBase)
	w.enableType(t, agentextension.TypeParallel, map[string]string{"apiKey": testAPIKey})

	_, err := w.svc.PlanWebWatch(t.Context(), w.request())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot watch the web on a schedule")
	assert.Empty(t, w.watcher.created)
}

func TestACheckRecordsEachSettledRunOnceAndHandsOnWhatIsNew(t *testing.T) {
	t.Parallel()

	w := newWatchFixture(t, testWebBase)
	view := w.set(t)
	w.watcher.runs = []agentextensionports.WebWatchRun{
		{
			ID: "run_3", Phase: agentextensionports.WebWatchRunPending,
			CreatedAt: testNow + 300,
		},
		completedRun("run_2", testNow+200, "javascript:alert(1)"),
		completedRun("run_1", testNow+100, "https://www.federalregister.gov/d/2026-1"),
	}

	check, err := w.svc.CheckWebWatch(t.Context(), w.tenant, view.ID)
	require.NoError(t, err)

	assert.False(t, check.Ended)
	require.Len(t, check.Undelivered, 1, "a run with no web page to show finds nothing")
	assert.Equal(t, "run_1", check.Undelivered[0].VendorRunID)
	assert.Equal(t, 2, check.Watch.RunCount, "the pending run is counted once it settles")
	assert.Equal(t, 1, check.Watch.FindingCount)
	assert.True(t, decimal.RequireFromString("0.03").Equal(check.Watch.CostUSD))
	require.NotNil(t, check.Watch.LastRunAt)
	assert.Equal(t, testNow+200, *check.Watch.LastRunAt)
	assert.True(t, decimal.RequireFromString("0.03").Equal(w.usage.cost))

	again, err := w.svc.CheckWebWatch(t.Context(), w.tenant, view.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, again.Watch.RunCount, "a run already counted is not counted again")
	assert.Equal(t, []int64{0, testNow + 200}, w.watcher.after,
		"the vendor is told which runs were already counted")
	assert.Len(t, w.findings.findings, 1)

	require.NoError(t, w.svc.MarkWebWatchDelivered(t.Context(), w.tenant, check.Undelivered))
	after, err := w.svc.CheckWebWatch(t.Context(), w.tenant, view.ID)
	require.NoError(t, err)
	assert.Empty(t, after.Undelivered)
}

func TestAWatchWhoseTimeRanOutEndsWithItsMonitorRemoved(t *testing.T) {
	t.Parallel()

	w := newWatchFixture(t, testWebBase)
	view := w.set(t)
	w.svc.now = func() int64 { return view.EndsAt }

	check, err := w.svc.CheckWebWatch(t.Context(), w.tenant, view.ID)
	require.NoError(t, err)

	assert.True(t, check.Ended)
	assert.Equal(t, extensionwebwatch.WebWatchStatusEnded, check.Watch.Status)
	assert.Equal(t, []string{"mon_1"}, w.watcher.deleted)
}

func TestAWatchTheVendorTurnedOffFailsAndSaysWhy(t *testing.T) {
	t.Parallel()

	w := newWatchFixture(t, testWebBase)
	view := w.set(t)
	w.watcher.disabled = true

	check, err := w.svc.CheckWebWatch(t.Context(), w.tenant, view.ID)
	require.NoError(t, err)

	assert.True(t, check.Ended)
	assert.Equal(t, extensionwebwatch.WebWatchStatusFailed, check.Watch.Status)
	assert.Contains(t, check.Watch.Failure, "Exa turned the watch off")
	assert.Equal(t, []string{"mon_1"}, w.watcher.deleted)
}

func TestAFailedRunIsBilledAndExplained(t *testing.T) {
	t.Parallel()

	w := newWatchFixture(t, testWebBase)
	view := w.set(t)
	w.watcher.runs = []agentextensionports.WebWatchRun{{
		ID:        "run_1",
		Phase:     agentextensionports.WebWatchRunFailed,
		Failure:   "insufficient_credits",
		CreatedAt: testNow + 100,
		SettledAt: testNow + 110,
		CostUSD:   extensionwebwatch.ExaMonitorRunCostUSD,
	}}

	check, err := w.svc.CheckWebWatch(t.Context(), w.tenant, view.ID)
	require.NoError(t, err)

	assert.False(t, check.Ended, "a failed run does not end the watch")
	assert.Contains(t, check.Watch.Failure, "ran out of credits")
	assert.Equal(t, 1, check.Watch.RunCount)
	assert.Equal(t, 1, w.usage.failures)
}

func TestOnlyWhoAWatchWasSetForStopsIt(t *testing.T) {
	t.Parallel()

	w := newWatchFixture(t, testWebBase)
	view := w.set(t)

	_, err := w.svc.StopWebWatch(t.Context(), &agentextensionports.StopWebWatchRequest{
		TenantInfo: w.tenant,
		WatchID:    view.ID,
		StoppedBy:  pulid.MustNew("usr_"),
	})
	require.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, w.watcher.deleted)

	stopped, err := w.svc.StopWebWatch(t.Context(), &agentextensionports.StopWebWatchRequest{
		TenantInfo: w.tenant,
		WatchID:    view.ID,
		StoppedBy:  w.tenant.UserID,
	})
	require.NoError(t, err)
	assert.Equal(t, extensionwebwatch.WebWatchStatusStopped, stopped.Status)
	assert.Equal(t, w.tenant.UserID, stopped.StoppedByID)
	assert.Equal(t, []string{"mon_1"}, w.watcher.deleted)
	assert.Equal(t, []string{webwatchcontract.WorkflowIDFor(view.ID) + "/" +
		webwatchcontract.SignalCheck}, w.workflows.signals)

	again, err := w.svc.StopWebWatch(t.Context(), &agentextensionports.StopWebWatchRequest{
		TenantInfo:  w.tenant,
		WatchID:     view.ID,
		Administers: true,
	})
	require.NoError(t, err)
	assert.Equal(t, extensionwebwatch.WebWatchStatusStopped, again.Status)
	assert.Len(t, w.watcher.deleted, 1, "a watch already stopped is not stopped twice")
}

func TestAWatchStaysRunningWhenTheVendorCannotRemoveIt(t *testing.T) {
	t.Parallel()

	w := newWatchFixture(t, testWebBase)
	view := w.set(t)
	w.watcher.deleteErr = errors.New("exa: 503")

	_, err := w.svc.StopWebWatch(t.Context(), &agentextensionports.StopWebWatchRequest{
		TenantInfo:  w.tenant,
		WatchID:     view.ID,
		Administers: true,
	})
	require.Error(t, err)
	assert.Equal(t, extensionwebwatch.WebWatchStatusActive, w.watches.only(t).Status,
		"a watch the vendor still runs is not recorded as stopped")
}

func TestADeliverySignedWithTheWatchsSecretAsksForACheck(t *testing.T) {
	t.Parallel()

	w := newWatchFixture(t, testWebBase)
	view := w.set(t)

	forged := http.Header{}
	forged.Set(testSignatureName, "not-the-secret")
	err := w.svc.ReceiveWebWatchDelivery(t.Context(),
		&agentextensionports.ReceiveWebWatchDeliveryRequest{
			TenantInfo: w.tenant,
			WatchID:    view.ID,
			Delivery:   agentextensionports.WebWatchDelivery{Header: forged},
		})
	require.Error(t, err)
	assert.Empty(t, w.workflows.signals)

	signed := http.Header{}
	signed.Set(testSignatureName, testWatchSecret)
	require.NoError(t, w.svc.ReceiveWebWatchDelivery(t.Context(),
		&agentextensionports.ReceiveWebWatchDeliveryRequest{
			TenantInfo: w.tenant,
			WatchID:    view.ID,
			Delivery:   agentextensionports.WebWatchDelivery{Header: signed},
		}))
	assert.Equal(t, []string{webwatchcontract.WorkflowIDFor(view.ID) + "/" +
		webwatchcontract.SignalCheck}, w.workflows.signals)
}
