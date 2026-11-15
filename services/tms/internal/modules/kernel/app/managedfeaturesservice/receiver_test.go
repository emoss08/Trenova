package managedfeaturesservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/app/managedfeaturesservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingObserver struct {
	seen []*managedfeatures.Replacement
	err  error
}

func (o *recordingObserver) PlanChanged(
	_ context.Context,
	replacement *managedfeatures.Replacement,
) error {
	o.seen = append(o.seen, replacement)
	return o.err
}

func newReceiver(
	repo *fakeRepo,
	observers ...managedfeaturesports.PlanObserver,
) *managedfeaturesservice.Receiver {
	svc, _ := newService(repo)
	return managedfeaturesservice.NewReceiver(managedfeaturesservice.ReceiverParams{
		Features:  svc,
		Observers: observers,
		Logger:    zap.NewNop(),
	})
}

func TestAChangedListWritesOneChangeAndTellsTheObservers(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{state: &managedfeatures.State{
		ID:       managedfeatures.CurrentStateID,
		Features: []platformcatalog.FeatureKey{"tms.dispatch_intelligence"},
	}}
	observer := &recordingObserver{}
	receiver := newReceiver(repo, observer)

	require.NoError(t, receiver.SetFeatures(t.Context(), []platformcatalog.FeatureKey{"ai.desk"}))

	require.Len(t, repo.changes, 1)
	assert.Equal(t, []platformcatalog.FeatureKey{"ai.desk"}, repo.changes[0].Added)
	assert.Equal(t, []platformcatalog.FeatureKey{"tms.dispatch_intelligence"}, repo.changes[0].Removed)
	assert.False(t, repo.changes[0].Starting)
	require.Len(t, observer.seen, 1)
	assert.Equal(t,
		[]platformcatalog.FeatureKey{"tms.dispatch_intelligence"},
		observer.seen[0].Previous.Features,
	)
}

func TestAnUnchangedListWritesNoChangeAndTellsNobody(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{state: &managedfeatures.State{
		ID:       managedfeatures.CurrentStateID,
		Features: []platformcatalog.FeatureKey{"ai.desk"},
	}}
	observer := &recordingObserver{}
	receiver := newReceiver(repo, observer)

	require.NoError(t, receiver.SetFeatures(t.Context(),
		[]platformcatalog.FeatureKey{"ai.desk", "tms.core"}))

	assert.Empty(t, repo.changes)
	assert.Empty(t, observer.seen)
}

func TestTheFirstListIsRecordedAsTheStartingPlan(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	observer := &recordingObserver{}
	receiver := newReceiver(repo, observer)

	require.NoError(t, receiver.SetFeatures(t.Context(), []platformcatalog.FeatureKey{"ai.desk"}))

	require.Len(t, repo.changes, 1)
	assert.True(t, repo.changes[0].Starting)
	require.Len(t, observer.seen, 1)
	assert.Nil(t, observer.seen[0].Previous)
}

func TestAFailingObserverDoesNotFailTheListAndTheChangeIsKept(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{state: &managedfeatures.State{
		ID:       managedfeatures.CurrentStateID,
		Features: []platformcatalog.FeatureKey{"ai.desk"},
	}}
	failing := &recordingObserver{err: errors.New("notifications are down")}
	after := &recordingObserver{}
	receiver := newReceiver(repo, failing, after)

	require.NoError(t, receiver.SetFeatures(t.Context(), nil))

	require.Len(t, repo.changes, 1)
	assert.Equal(t, []platformcatalog.FeatureKey{}, repo.state.Features)
	assert.Len(t, after.seen, 1, "the observers after a failing one are still told")
}

func TestAStoreFailureIsReturnedAndNobodyIsTold(t *testing.T) {
	t.Parallel()

	down := errors.New("database down")
	observer := &recordingObserver{}
	receiver := newReceiver(&fakeRepo{err: down}, observer)

	require.ErrorIs(t, receiver.SetFeatures(t.Context(), nil), down)
	assert.Empty(t, observer.seen)
}
