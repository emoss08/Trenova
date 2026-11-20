package managedfeaturesservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/app/managedfeaturesservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
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

func received(features ...platformcatalog.FeatureKey) *managedfeatures.Received {
	return &managedfeatures.Received{Features: features}
}

func newReceiverIn(
	repo *fakeRepo,
	shared bool,
	observers ...managedfeaturesports.PlanObserver,
) *managedfeaturesservice.Receiver {
	svc, _ := newServiceIn(repo, shared)
	return managedfeaturesservice.NewReceiver(managedfeaturesservice.ReceiverParams{
		Features:  svc,
		Observers: observers,
		Logger:    zap.NewNop(),
	})
}

func newReceiver(
	repo *fakeRepo,
	observers ...managedfeaturesports.PlanObserver,
) *managedfeaturesservice.Receiver {
	return newReceiverIn(repo, false, observers...)
}

func setInstance(
	t *testing.T,
	receiver *managedfeaturesservice.Receiver,
	in *managedfeatures.Received,
) error {
	t.Helper()
	return receiver.SetFeatures(t.Context(), managedscope.Instance(), in)
}

func TestAChangedListWritesOneChangeAndTellsTheObservers(t *testing.T) {
	t.Parallel()

	repo := newRepo(instanceListed("tms.dispatch_intelligence"))
	observer := &recordingObserver{}
	receiver := newReceiver(repo, observer)

	require.NoError(t, setInstance(t, receiver, received("ai.desk")))

	require.Len(t, repo.changes, 1)
	assert.Equal(t, []platformcatalog.FeatureKey{"ai.desk"}, repo.changes[0].Added)
	assert.Equal(
		t,
		[]platformcatalog.FeatureKey{"tms.dispatch_intelligence"},
		repo.changes[0].Removed,
	)
	assert.False(t, repo.changes[0].Starting)
	assert.Equal(t, managedscope.InstanceKey, repo.changes[0].ScopeKey)
	require.Len(t, observer.seen, 1)
	assert.True(t, observer.seen[0].Scope.IsInstance())
	assert.Equal(t,
		[]platformcatalog.FeatureKey{"tms.dispatch_intelligence"},
		observer.seen[0].Previous.Features,
	)
}

func TestAnUnchangedListWritesNoChangeAndTellsNobody(t *testing.T) {
	t.Parallel()

	repo := newRepo(instanceListed("ai.desk"))
	observer := &recordingObserver{}
	receiver := newReceiver(repo, observer)

	require.NoError(t, setInstance(t, receiver, received("ai.desk", "tms.core")))

	assert.Empty(t, repo.changes)
	assert.Empty(t, observer.seen)
}

func TestTheFirstListIsRecordedAsTheStartingPlan(t *testing.T) {
	t.Parallel()

	repo := newRepo()
	observer := &recordingObserver{}
	receiver := newReceiver(repo, observer)

	require.NoError(t, setInstance(t, receiver, received("ai.desk")))

	require.Len(t, repo.changes, 1)
	assert.True(t, repo.changes[0].Starting)
	require.Len(t, observer.seen, 1)
	assert.Nil(t, observer.seen[0].Previous)
}

func TestAFailingObserverDoesNotFailTheListAndTheChangeIsKept(t *testing.T) {
	t.Parallel()

	repo := newRepo(instanceListed("ai.desk"))
	failing := &recordingObserver{err: errors.New("notifications are down")}
	after := &recordingObserver{}
	receiver := newReceiver(repo, failing, after)

	require.NoError(t, setInstance(t, receiver, received()))

	require.Len(t, repo.changes, 1)
	assert.Equal(t, []platformcatalog.FeatureKey{}, repo.instance().Features)
	assert.Len(t, after.seen, 1, "the observers after a failing one are still told")
}

func TestAStoreFailureIsReturnedAndNobodyIsTold(t *testing.T) {
	t.Parallel()

	down := errors.New("database down")
	observer := &recordingObserver{}
	repo := newRepo()
	repo.err = down
	receiver := newReceiver(repo, observer)

	require.ErrorIs(t, setInstance(t, receiver, received()), down)
	assert.Empty(t, observer.seen)
}

func TestAnEndingOnlyChangeIsRecordedAndToldWithTheFeaturesUnchanged(t *testing.T) {
	t.Parallel()

	repo := newRepo(instanceListed("tms.accounting"))
	observer := &recordingObserver{}
	receiver := newReceiver(repo, observer)
	endings := []managedfeatures.AddOnEnding{
		{AddOn: platformcatalog.AddOnAccounting, EndsAt: now + 86_400},
	}

	require.NoError(t, setInstance(t, receiver, &managedfeatures.Received{
		Features:     []platformcatalog.FeatureKey{"tms.accounting"},
		AddOnEndings: &endings,
	}))

	require.Len(t, repo.changes, 1)
	assert.Equal(t, endings, repo.changes[0].Ending)
	require.Len(t, observer.seen, 1, "every change to the plan is told, so open pages update")
	assert.False(t, observer.seen[0].FeaturesChanged(), "but the features did not change")
}

func endingOn(scope managedscope.Scope, ends int64) *managedfeatures.State {
	state := listed(scope)
	state.PlanEndsAt = ends
	state.PlanEndAnnounced = ends
	return state
}

func TestTheReminderIsSentOnceThreeDaysBeforeTheEnd(t *testing.T) {
	t.Parallel()

	ends := now + 2*86_400
	repo := newRepo(endingOn(managedscope.Instance(), ends))
	observer := &recordingObserver{}
	receiver := newReceiver(repo, observer)

	first, err := receiver.RemindPlanEnd(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, first.Reminded)
	require.Len(t, observer.seen, 1)
	assert.Equal(t, managedfeatures.PlanEndReminder, observer.seen[0].PlanEnd.Kind)
	assert.Equal(t, ends, observer.seen[0].PlanEnd.EndsAt)

	again, err := receiver.RemindPlanEnd(t.Context())
	require.NoError(t, err)
	assert.Zero(t, again.Reminded, "a reminder is sent once")
	assert.Len(t, observer.seen, 1)
}

func TestNoReminderWhileThePlanIsNotEnding(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	receiver := newReceiver(newRepo(instanceListed()), observer)

	result, err := receiver.RemindPlanEnd(t.Context())
	require.NoError(t, err)
	assert.Zero(t, result.Reminded)
	assert.Empty(t, observer.seen)

	result, err = newReceiver(newRepo(), observer).RemindPlanEnd(t.Context())
	require.NoError(t, err)
	assert.Zero(t, result.Reminded, "an instance that never received a list has nothing to remind")
}

func TestEachOrganizationWhosePlanEndsIsRemindedOnItsOwn(t *testing.T) {
	t.Parallel()

	ending, later, suspended := tenant(), tenant(), tenant()
	suspendedState := endingOn(managedscope.Organization(suspended), now+86_400)
	suspendedState.Lifecycle = managedfeatures.LifecycleSuspended
	repo := newRepo(
		endingOn(managedscope.Organization(ending), now+86_400),
		endingOn(managedscope.Organization(later), now+30*86_400),
		suspendedState,
	)
	observer := &recordingObserver{}
	receiver := newReceiverIn(repo, true, observer)

	result, err := receiver.RemindPlanEnd(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Reminded)
	require.Len(t, observer.seen, 1)
	scoped, ok := observer.seen[0].Scope.Tenant()
	require.True(t, ok)
	assert.Equal(t, ending.OrganizationID, scoped.OrganizationID)
	assert.Equal(t, ending.BusinessUnitID, scoped.BusinessUnitID)
}

func TestALifecycleChangeIsToldToTheObservers(t *testing.T) {
	t.Parallel()

	org := tenant()
	repo := newRepo(listed(managedscope.Organization(org), "tms.accounting"))
	observer := &recordingObserver{}
	receiver := newReceiverIn(repo, true, observer)

	require.NoError(t, receiver.SetLifecycle(
		t.Context(),
		managedscope.Organization(org),
		managedfeatures.LifecycleSuspended,
	))
	require.Len(t, observer.seen, 1)
	assert.True(t, observer.seen[0].LifecycleChanged())
	assert.False(t, observer.seen[0].FeaturesChanged())

	require.NoError(t, receiver.SetLifecycle(
		t.Context(),
		managedscope.Organization(org),
		managedfeatures.LifecycleSuspended,
	))
	assert.Len(t, observer.seen, 1, "the same lifecycle again tells nobody")
}
