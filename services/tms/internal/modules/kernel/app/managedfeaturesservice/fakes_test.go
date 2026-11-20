package managedfeaturesservice_test

import (
	"context"
	"slices"
	"sort"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/app/managedfeaturesservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

const now = int64(1_794_650_400)

type fakeRepo struct {
	mu        sync.Mutex
	states    map[string]*managedfeatures.State
	gets      map[string]int
	saves     int
	anywheres int
	changes   []*managedfeatures.Change
	err       error
}

func newRepo(states ...*managedfeatures.State) *fakeRepo {
	repo := &fakeRepo{
		states: make(map[string]*managedfeatures.State, len(states)),
		gets:   map[string]int{},
	}
	for _, state := range states {
		repo.states[state.ScopeKey] = state
	}
	return repo
}

func (f *fakeRepo) state(scope managedscope.Scope) *managedfeatures.State {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.states[scope.Key()]
}

func (f *fakeRepo) instance() *managedfeatures.State {
	return f.state(managedscope.Instance())
}

func (f *fakeRepo) heldLocked(scope managedscope.Scope) *managedfeatures.State {
	held := f.states[scope.Key()]
	if held == nil {
		return nil
	}
	c := *held
	c.Features = slices.Clone(held.Features)
	return &c
}

func (f *fakeRepo) store(state *managedfeatures.State) {
	c := *state
	c.Features = slices.Clone(state.Features)
	f.states[state.ScopeKey] = &c
	f.saves++
}

func (f *fakeRepo) Get(
	_ context.Context,
	scope managedscope.Scope,
) (*managedfeatures.State, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gets[scope.Key()]++
	if f.err != nil {
		return nil, f.err
	}
	held := f.heldLocked(scope)
	if held == nil {
		return nil, managedfeatures.ErrNoState
	}
	return held, nil
}

func (f *fakeRepo) Replace(
	_ context.Context,
	req *managedfeaturesports.ReplaceRequest,
) (*managedfeatures.Replacement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	replacement := managedfeatures.Replace(
		req.Scope,
		f.heldLocked(req.Scope),
		req.Received,
		req.Now,
	)
	if !replacement.Updated() {
		return replacement, nil
	}
	f.store(replacement.Current)
	if replacement.Changed() {
		f.changes = append(f.changes, replacement.Change)
	}
	return replacement, nil
}

func (f *fakeRepo) SetLifecycle(
	_ context.Context,
	req *managedfeaturesports.LifecycleRequest,
) (*managedfeatures.Replacement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	replacement := managedfeatures.SetLifecycle(req.Scope, f.heldLocked(req.Scope), req.Lifecycle)
	if replacement.Updated() {
		f.store(replacement.Current)
	}
	return replacement, nil
}

func (f *fakeRepo) DuePlanEndReminders(
	_ context.Context,
	req *managedfeaturesports.DueRemindersRequest,
) ([]managedscope.Scope, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	keys := make([]string, 0, len(f.states))
	for key := range f.states {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	due := make([]managedscope.Scope, 0)
	for _, key := range keys {
		if state := f.states[key]; state.PlanEndReminderDue(req.Now) {
			due = append(due, state.Scope())
		}
	}
	return due, nil
}

func (f *fakeRepo) RemindPlanEnd(
	_ context.Context,
	scope managedscope.Scope,
	at int64,
) (*managedfeatures.Replacement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	replacement := managedfeatures.RemindPlanEnd(f.heldLocked(scope), at)
	if replacement.Updated() {
		f.store(replacement.Current)
	}
	return replacement, nil
}

func (f *fakeRepo) History(
	_ context.Context,
	scope managedscope.Scope,
	_ int,
) (*managedfeatures.State, []*managedfeatures.Change, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	changes := make([]*managedfeatures.Change, 0, len(f.changes))
	for _, change := range f.changes {
		if change.ScopeKey == scope.Key() {
			changes = append(changes, change)
		}
	}
	return f.heldLocked(scope), changes, nil
}

func (f *fakeRepo) AnyActiveScope(
	_ context.Context,
	feature *platformcatalog.FeatureKey,
) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.anywheres++
	if f.err != nil {
		return false, f.err
	}
	for key, state := range f.states {
		if key == managedscope.InstanceKey || state.Suspended() || !state.HasList() {
			continue
		}
		if feature == nil || slices.Contains(state.Features, *feature) {
			return true, nil
		}
	}
	return false, nil
}

func listed(
	scope managedscope.Scope,
	features ...platformcatalog.FeatureKey,
) *managedfeatures.State {
	return &managedfeatures.State{
		ScopeKey:            scope.Key(),
		ScopeBusinessUnitID: scope.BusinessUnitID(),
		Lifecycle:           managedfeatures.LifecycleActive,
		Features:            features,
		ReceivedAt:          now - 3_600,
	}
}

func instanceListed(features ...platformcatalog.FeatureKey) *managedfeatures.State {
	return listed(managedscope.Instance(), features...)
}

func tenant() dbscope.Tenant {
	return dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func newServiceIn(
	repo *fakeRepo,
	shared bool,
) (*managedfeaturesservice.Service, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.InfoLevel)
	return managedfeaturesservice.New(managedfeaturesservice.Params{
		Repo:   repo,
		Scopes: managedscope.NewResolver(shared),
		Clock:  func() int64 { return now },
		Logger: zap.New(core),
	}), logs
}

func newService(repo *fakeRepo) (*managedfeaturesservice.Service, *observer.ObservedLogs) {
	return newServiceIn(repo, false)
}

func setFeatures(
	t *testing.T,
	svc *managedfeaturesservice.Service,
	features []platformcatalog.FeatureKey,
) *managedfeatures.Replacement {
	t.Helper()
	replacement, err := svc.SetFeatures(
		t.Context(),
		managedscope.Instance(),
		&managedfeatures.Received{Features: features},
	)
	require.NoError(t, err)
	return replacement
}

func enabledFor(
	t *testing.T,
	svc *managedfeaturesservice.Service,
	of dbscope.Tenant,
	key platformcatalog.FeatureKey,
) bool {
	t.Helper()
	on, err := svc.Enabled(dbscope.WithTenant(t.Context(), of), key)
	require.NoError(t, err)
	return on
}

func enabled(
	t *testing.T,
	svc *managedfeaturesservice.Service,
	key platformcatalog.FeatureKey,
) bool {
	t.Helper()
	on, err := svc.Enabled(t.Context(), key)
	require.NoError(t, err)
	return on
}
