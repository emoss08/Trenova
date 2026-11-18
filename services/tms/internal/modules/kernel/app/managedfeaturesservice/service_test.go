package managedfeaturesservice_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/app/managedfeaturesservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

const now = int64(1_794_650_400)

type fakeRepo struct {
	mu      sync.Mutex
	state   *managedfeatures.State
	gets    int
	saves   int
	changes []*managedfeatures.Change
	err     error
}

func (f *fakeRepo) Get(context.Context) (*managedfeatures.State, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gets++
	if f.err != nil {
		return nil, f.err
	}
	if f.state == nil {
		return nil, managedfeatures.ErrNoState
	}
	c := *f.state
	c.Features = slices.Clone(f.state.Features)
	return &c, nil
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
	replacement := managedfeatures.Replace(f.state, req.Received, req.Now)
	if !replacement.Updated() {
		return replacement, nil
	}
	f.saves++
	c := *replacement.Current
	c.Features = slices.Clone(replacement.Current.Features)
	f.state = &c
	if replacement.Changed() {
		f.changes = append(f.changes, replacement.Change)
	}
	return replacement, nil
}

func (f *fakeRepo) RemindPlanEnd(
	_ context.Context,
	at int64,
) (*managedfeatures.Replacement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	replacement := managedfeatures.RemindPlanEnd(f.state, at)
	if !replacement.Updated() {
		return replacement, nil
	}
	f.saves++
	c := *replacement.Current
	f.state = &c
	return replacement, nil
}

func (f *fakeRepo) History(
	context.Context,
	int,
) (*managedfeatures.State, []*managedfeatures.Change, error) {
	return f.state, f.changes, nil
}

func setFeatures(
	t *testing.T,
	svc *managedfeaturesservice.Service,
	features []platformcatalog.FeatureKey,
) *managedfeatures.Replacement {
	t.Helper()
	replacement, err := svc.SetFeatures(t.Context(), &managedfeatures.Received{Features: features})
	require.NoError(t, err)
	return replacement
}

func newService(repo *fakeRepo) (*managedfeaturesservice.Service, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.InfoLevel)
	return managedfeaturesservice.New(managedfeaturesservice.Params{
		Repo:   repo,
		Clock:  func() int64 { return now },
		Logger: zap.New(core),
	}), logs
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

func TestTheGateWithNoList(t *testing.T) {
	t.Parallel()

	svc, _ := newService(&fakeRepo{})
	assert.True(t, enabled(t, svc, platformcatalog.FeatureCoreTMS), "a base feature is on")
	assert.False(t, enabled(t, svc, platformcatalog.FeatureAccounting), "an add-on is off")
}

func TestTheGateWithAList(t *testing.T) {
	t.Parallel()

	svc, _ := newService(&fakeRepo{state: &managedfeatures.State{
		ID:       managedfeatures.CurrentStateID,
		Features: []platformcatalog.FeatureKey{platformcatalog.FeatureAccounting},
	}})
	assert.True(t, enabled(t, svc, platformcatalog.FeatureAccounting), "a listed add-on is on")
	assert.False(t, enabled(t, svc, platformcatalog.FeatureAIDesk), "an unlisted add-on is off")
	assert.True(t, enabled(t, svc, platformcatalog.FeatureDispatch), "a base feature is on")
	assert.False(t, enabled(t, svc, "tms.teleport"), "an unknown key is off")
}

func TestTheGateSurfacesARepositoryError(t *testing.T) {
	t.Parallel()

	down := errors.New("database down")
	svc, _ := newService(&fakeRepo{err: down})
	_, err := svc.Enabled(t.Context(), platformcatalog.FeatureAccounting)
	require.ErrorIs(t, err, down)
}

func TestTheListIsCached(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc, _ := newService(repo)
	for range 5 {
		_, err := svc.Enabled(t.Context(), platformcatalog.FeatureAccounting)
		require.NoError(t, err)
	}
	assert.Equal(t, 1, repo.gets)
}

func TestSetFeaturesStoresTheNormalisedList(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc, logs := newService(repo)
	setFeatures(t, svc, []platformcatalog.FeatureKey{
		"tms.rating", "tms.core", "ai.desk", "tms.teleport", "ai.desk",
	})

	require.NotNil(t, repo.state)
	assert.Equal(t, managedfeatures.CurrentStateID, repo.state.ID)
	assert.Equal(t, []platformcatalog.FeatureKey{"ai.desk", "tms.rating"}, repo.state.Features)
	assert.Equal(t, now, repo.state.ReceivedAt)

	entries := logs.All()
	require.Len(t, entries, 1)
	assert.Equal(t, zapcore.WarnLevel, entries[0].Level)
	fields := entries[0].ContextMap()
	assert.Equal(t, int64(2), fields["features"])
	assert.Equal(t, []any{"ai.desk", "tms.rating"}, fields["added"])
	assert.Equal(t, []any{}, fields["removed"])
	assert.Equal(t, []any{"tms.teleport"}, fields["ignored"])
}

func TestSetFeaturesStoresAnEmptyList(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc, _ := newService(repo)
	setFeatures(t, svc, nil)
	require.NotNil(t, repo.state)
	assert.Equal(t, []platformcatalog.FeatureKey{}, repo.state.Features)
}

func TestSetFeaturesWritesNothingWhenTheListIsUnchanged(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc, logs := newService(repo)
	list := []platformcatalog.FeatureKey{"tms.accounting", "ai.desk"}
	setFeatures(t, svc, list)
	setFeatures(t, svc,
		[]platformcatalog.FeatureKey{"ai.desk", "tms.accounting", "tms.core"})

	assert.Equal(t, 1, repo.saves)
	assert.Equal(t, 1, logs.Len())
}

func TestSetFeaturesStoresAChangeAndTheGateSeesItAtOnce(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc, logs := newService(repo)
	setFeatures(t, svc,
		[]platformcatalog.FeatureKey{"tms.accounting", "ai.desk"})
	assert.True(t, enabled(t, svc, platformcatalog.FeatureAccounting))

	setFeatures(t, svc,
		[]platformcatalog.FeatureKey{"ai.desk", "tms.rating"})
	assert.Equal(t, 2, repo.saves)
	assert.Equal(t, []platformcatalog.FeatureKey{"ai.desk", "tms.rating"}, repo.state.Features)
	assert.False(t, enabled(t, svc, platformcatalog.FeatureAccounting), "the cache was dropped")
	assert.True(t, enabled(t, svc, platformcatalog.FeatureRating))

	entries := logs.All()
	require.Len(t, entries, 2)
	assert.Equal(t, zapcore.InfoLevel, entries[1].Level)
	fields := entries[1].ContextMap()
	assert.Equal(t, int64(2), fields["features"])
	assert.Equal(t, []any{"tms.rating"}, fields["added"])
	assert.Equal(t, []any{"tms.accounting"}, fields["removed"])
	assert.NotContains(t, fields, "ignored")
}

func TestSetFeaturesSurfacesARepositoryError(t *testing.T) {
	t.Parallel()

	down := errors.New("database down")
	repo := &fakeRepo{err: down}
	svc, _ := newService(repo)
	_, err := svc.SetFeatures(t.Context(), &managedfeatures.Received{})
	require.ErrorIs(t, err, down)
	assert.Equal(t, 0, repo.saves)
}

func TestStateIsNilBeforeAList(t *testing.T) {
	t.Parallel()

	svc, _ := newService(&fakeRepo{})
	state, err := svc.State(t.Context())
	require.NoError(t, err)
	assert.Nil(t, state)
}

func TestSetFeaturesRefusesABadPlanURLBeforeWritingAnything(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc, _ := newService(repo)
	bad := "javascript:alert(1)"
	_, err := svc.SetFeatures(t.Context(), &managedfeatures.Received{PlanURL: &bad})

	_, refused := errortypes.RefusalOf(err)
	require.True(t, refused)
	assert.Equal(t, 0, repo.saves)
	assert.Nil(t, repo.state)
}

func TestSetFeaturesStoresThePlanURLAndEndingsAndKeepsThemWhenLeftOut(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc, _ := newService(repo)
	planURL := "https://console.trenova.cloud/instances/1/plan"
	endings := []managedfeatures.AddOnEnding{
		{AddOn: platformcatalog.AddOnAccounting, EndsAt: now + 86_400},
	}
	_, err := svc.SetFeatures(t.Context(), &managedfeatures.Received{
		Features:     []platformcatalog.FeatureKey{"tms.accounting"},
		PlanURL:      &planURL,
		AddOnEndings: &endings,
	})
	require.NoError(t, err)

	setFeatures(t, svc, []platformcatalog.FeatureKey{"tms.accounting"})

	assert.Equal(t, 1, repo.saves, "a request without them changes nothing")
	state, err := svc.State(t.Context())
	require.NoError(t, err)
	assert.Equal(t, planURL, state.PlanURL)
	assert.Equal(t, endings, state.AddOnEndings)
	assert.Equal(t, int64(now+86_400), state.EndsAt(platformcatalog.AddOnAccounting))
}

func TestAnUnknownAddOnIsIgnoredAndLoggedAtMostOnceAnHour(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc, logs := newService(repo)
	endings := []managedfeatures.AddOnEnding{{AddOn: "ZZZ", EndsAt: now + 86_400}}
	received := &managedfeatures.Received{
		Features:     []platformcatalog.FeatureKey{"tms.accounting"},
		AddOnEndings: &endings,
	}
	for range 3 {
		_, err := svc.SetFeatures(t.Context(), received)
		require.NoError(t, err)
	}

	assert.Empty(t, repo.state.AddOnEndings)
	entries := logs.All()
	require.Len(t, entries, 2, "the first write, then one reminder for the repeats")
	for _, entry := range entries {
		assert.Equal(t, zapcore.WarnLevel, entry.Level)
		assert.Equal(t, []any{"ZZZ"}, entry.ContextMap()["ignoredAddOns"])
	}
}
