//go:build integration

package managedfeaturesrepository

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newRepo(t *testing.T) (context.Context, *keel.DB, managedfeaturesports.StateRepository) {
	t.Helper()
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	return ctx, db, New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
}

var instance = managedscope.Instance()

func organization() managedscope.Scope {
	return managedscope.Organization(dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

func replace(
	t *testing.T,
	ctx context.Context,
	repo managedfeaturesports.StateRepository,
	now int64,
	features ...platformcatalog.FeatureKey,
) *managedfeatures.Replacement {
	t.Helper()
	return replaceIn(t, ctx, repo, instance, now, features...)
}

func replaceIn(
	t *testing.T,
	ctx context.Context,
	repo managedfeaturesports.StateRepository,
	scope managedscope.Scope,
	now int64,
	features ...platformcatalog.FeatureKey,
) *managedfeatures.Replacement {
	t.Helper()
	replacement, err := repo.Replace(ctx, &managedfeaturesports.ReplaceRequest{
		Scope:    scope,
		Received: &managedfeatures.Received{Features: features},
		Now:      now,
	})
	require.NoError(t, err)
	return replacement
}

func TestAChangedListWritesOneChangeRowAndAnUnchangedOneWritesNone(t *testing.T) {
	t.Parallel()

	ctx, _, repo := newRepo(t)

	first := replace(t, ctx, repo, 100, "ai.desk")
	require.True(t, first.Changed())
	assert.True(t, first.Change.Starting)

	same := replace(t, ctx, repo, 200, "ai.desk", "tms.core")
	assert.False(t, same.Changed())

	changed := replace(t, ctx, repo, 300, "tms.accounting")
	require.True(t, changed.Changed())

	held, changes, err := repo.History(ctx, instance, 20)
	require.NoError(t, err)
	assert.Equal(t, []platformcatalog.FeatureKey{"tms.accounting"}, held.Features)
	assert.Equal(t, int64(300), held.ReceivedAt)
	require.Len(t, changes, 2)
	assert.Equal(t, int64(300), changes[0].ChangedAt)
	assert.Equal(t, []platformcatalog.FeatureKey{"tms.accounting"}, changes[0].Added)
	assert.Equal(t, []platformcatalog.FeatureKey{"ai.desk"}, changes[0].Removed)
	assert.False(t, changes[0].Starting)
	assert.True(t, changes[1].Starting)
}

func TestTheChangeRowIsWrittenInTheSameTransactionAsTheList(t *testing.T) {
	t.Parallel()

	ctx, db, repo := newRepo(t)
	rollback := errors.New("roll back")

	err := dbtx.WriteErr(
		dbscope.WithSystem(ctx, "test the managed feature list write"),
		postgres.WrapDB(db),
		func(ctx context.Context) error {
			if _, replaceErr := repo.Replace(ctx, &managedfeaturesports.ReplaceRequest{
				Scope: instance,
				Received: &managedfeatures.Received{
					Features: []platformcatalog.FeatureKey{"ai.desk"},
				},
				Now: 100,
			}); replaceErr != nil {
				return replaceErr
			}
			return rollback
		},
	)
	require.ErrorIs(t, err, rollback)

	held, changes, err := repo.History(ctx, instance, 20)
	require.NoError(t, err)
	assert.Nil(t, held, "the list rolled back")
	assert.Empty(t, changes, "and its change row with it")
}

func TestTheChangeHistoryIsPrunedToTheNewest200(t *testing.T) {
	t.Parallel()

	ctx, _, repo := newRepo(t)
	lists := [][]platformcatalog.FeatureKey{{"ai.desk"}, {"tms.accounting"}}
	const writes = managedfeatures.ChangeHistoryLimit + 5
	for i := range writes {
		replace(t, ctx, repo, int64(1_000+i), lists[i%2]...)
	}

	_, changes, err := repo.History(ctx, instance, managedfeatures.ChangeHistoryLimit+50)
	require.NoError(t, err)
	require.Len(t, changes, managedfeatures.ChangeHistoryLimit)
	assert.Equal(t, int64(1_000+writes-1), changes[0].ChangedAt, "the newest is kept")
	assert.Equal(t,
		int64(1_000+writes-managedfeatures.ChangeHistoryLimit),
		changes[len(changes)-1].ChangedAt,
		"the oldest kept is the 200th newest",
	)
}

func TestThePlanURLAndEndingsAreStoredWithTheListAndAnEndingIsAChange(t *testing.T) {
	t.Parallel()

	ctx, _, repo := newRepo(t)
	replace(t, ctx, repo, 100, "tms.accounting")

	planURL := "https://console.trenova.cloud/instances/1/plan"
	endings := []managedfeatures.AddOnEnding{
		{AddOn: platformcatalog.AddOnAccounting, EndsAt: 1_797_206_400},
	}
	set, err := repo.Replace(ctx, &managedfeaturesports.ReplaceRequest{
		Scope: instance,
		Received: &managedfeatures.Received{
			Features:     []platformcatalog.FeatureKey{"tms.accounting"},
			PlanURL:      &planURL,
			AddOnEndings: &endings,
		},
		Now: 200,
	})
	require.NoError(t, err)
	require.True(t, set.Changed())

	same := replace(t, ctx, repo, 300, "tms.accounting")
	assert.False(t, same.Updated(), "a request without them keeps them")

	held, changes, err := repo.History(ctx, instance, 20)
	require.NoError(t, err)
	assert.Equal(t, planURL, held.PlanURL)
	assert.Equal(t, endings, held.AddOnEndings)
	require.Len(t, changes, 2)
	assert.Equal(t, endings, changes[0].Ending)
	assert.Empty(t, changes[0].Added)

	called := []managedfeatures.AddOnEnding{}
	kept, err := repo.Replace(ctx, &managedfeaturesports.ReplaceRequest{
		Scope: instance,
		Received: &managedfeatures.Received{
			Features:     []platformcatalog.FeatureKey{"tms.accounting"},
			AddOnEndings: &called,
		},
		Now: 400,
	})
	require.NoError(t, err)
	require.True(t, kept.Changed())

	held, changes, err = repo.History(ctx, instance, 20)
	require.NoError(t, err)
	assert.Empty(t, held.AddOnEndings)
	assert.Equal(t, planURL, held.PlanURL)
	require.Len(t, changes, 3)
	assert.Equal(t, []platformcatalog.AddOnKey{platformcatalog.AddOnAccounting}, changes[0].Kept)
}

func TestThePlansEndIsStoredRemindedOnceAndClearedWhenThePlanIsKept(t *testing.T) {
	t.Parallel()

	ctx, _, repo := newRepo(t)
	const (
		day = int64(86_400)
		now = int64(1_794_650_400)
		end = now + 30*day
	)
	ending := func(endsAt, at int64) *managedfeatures.Replacement {
		t.Helper()
		replacement, err := repo.Replace(ctx, &managedfeaturesports.ReplaceRequest{
			Scope: instance,
			Received: &managedfeatures.Received{
				Features:   []platformcatalog.FeatureKey{"tms.accounting"},
				PlanEndsAt: endsAt,
			},
			Now: at,
		})
		require.NoError(t, err)
		return replacement
	}
	remind := func(at int64) *managedfeatures.Replacement {
		t.Helper()
		replacement, err := repo.RemindPlanEnd(ctx, instance, at)
		require.NoError(t, err)
		return replacement
	}

	none := remind(now)
	assert.Nil(t, none.PlanEnd, "nothing to remind of before any list arrived")

	set := ending(end, now)
	require.NotNil(t, set.PlanEnd)
	assert.Equal(t, managedfeatures.PlanEndSet, set.PlanEnd.Kind)
	assert.Nil(t, ending(end, now+60).PlanEnd, "the same end again announces nothing")

	assert.Nil(t, remind(now+day).PlanEnd, "too early to remind")
	due := remind(end - 2*day)
	require.NotNil(t, due.PlanEnd)
	assert.Equal(t, managedfeatures.PlanEndReminder, due.PlanEnd.Kind)
	assert.Nil(t, remind(end-day).PlanEnd, "reminded once")

	held, err := repo.Get(ctx, instance)
	require.NoError(t, err)
	assert.Equal(t, end, held.PlanEndsAt)
	assert.Equal(t, end, held.PlanEndAnnounced)
	assert.Equal(t, end, held.PlanEndReminded)

	kept := ending(0, end-day+60)
	require.NotNil(t, kept.PlanEnd)
	assert.Equal(t, managedfeatures.PlanEndKept, kept.PlanEnd.Kind)

	held, err = repo.Get(ctx, instance)
	require.NoError(t, err)
	assert.Zero(t, held.PlanEndsAt)
	assert.Zero(t, held.PlanEndAnnounced)
	assert.Zero(t, held.PlanEndReminded)
}

func TestEachScopeKeepsItsOwnPlanAndHistory(t *testing.T) {
	t.Parallel()

	ctx, _, repo := newRepo(t)
	a, b := organization(), organization()
	replaceIn(t, ctx, repo, a, 100, "tms.accounting")
	replaceIn(t, ctx, repo, b, 100, "ai.desk")
	replaceIn(t, ctx, repo, b, 200, "ai.desk", "tms.rating")

	heldA, changesA, err := repo.History(ctx, a, 20)
	require.NoError(t, err)
	assert.Equal(t, []platformcatalog.FeatureKey{"tms.accounting"}, heldA.Features)
	assert.Equal(t, a.BusinessUnitID(), heldA.ScopeBusinessUnitID)
	assert.Equal(t, managedfeatures.LifecycleActive, heldA.Lifecycle)
	require.Len(t, changesA, 1)

	heldB, changesB, err := repo.History(ctx, b, 20)
	require.NoError(t, err)
	assert.Equal(t, []platformcatalog.FeatureKey{"ai.desk", "tms.rating"}, heldB.Features)
	require.Len(t, changesB, 2)
	for _, change := range changesB {
		assert.Equal(t, b.Key(), change.ScopeKey)
	}

	_, err = repo.Get(ctx, organization())
	require.ErrorIs(t, err, managedfeatures.ErrNoState)
}

func TestPruningKeepsTheNewest200OfEachScope(t *testing.T) {
	t.Parallel()

	ctx, _, repo := newRepo(t)
	busy, quiet := organization(), organization()
	replaceIn(t, ctx, repo, quiet, 10, "ai.desk")
	lists := [][]platformcatalog.FeatureKey{{"ai.desk"}, {"tms.accounting"}}
	for i := range managedfeatures.ChangeHistoryLimit + 3 {
		replaceIn(t, ctx, repo, busy, int64(1_000+i), lists[i%2]...)
	}

	_, busyChanges, err := repo.History(ctx, busy, managedfeatures.ChangeHistoryLimit+50)
	require.NoError(t, err)
	assert.Len(t, busyChanges, managedfeatures.ChangeHistoryLimit)
	_, quietChanges, err := repo.History(ctx, quiet, 20)
	require.NoError(t, err)
	assert.Len(t, quietChanges, 1, "another scope's pruning keeps this one's history")
}

func TestALifecycleIsStoredPerScopeAndAnsweredByAnyActiveScope(t *testing.T) {
	t.Parallel()

	ctx, _, repo := newRepo(t)
	withAccounting, withDesk := organization(), organization()
	replaceIn(t, ctx, repo, withAccounting, 100, "tms.accounting")
	replaceIn(t, ctx, repo, withDesk, 100, "ai.desk")
	replace(t, ctx, repo, 100, "tms.fuel_tax")

	accounting := platformcatalog.FeatureAccounting
	fuel := platformcatalog.FeatureFuelTax
	anywhere := func(feature *platformcatalog.FeatureKey) bool {
		t.Helper()
		on, err := repo.AnyActiveScope(ctx, feature)
		require.NoError(t, err)
		return on
	}
	assert.True(t, anywhere(&accounting))
	assert.True(t, anywhere(nil))
	assert.False(t, anywhere(&fuel), "the instance scope is not an organization on a shared server")

	suspended, err := repo.SetLifecycle(ctx, &managedfeaturesports.LifecycleRequest{
		Scope:     withAccounting,
		Lifecycle: managedfeatures.LifecycleSuspended,
	})
	require.NoError(t, err)
	require.True(t, suspended.Updated())
	assert.False(t, anywhere(&accounting), "a suspended organization counts for nothing")
	assert.True(t, anywhere(nil), "another organization is still active")

	held, err := repo.Get(ctx, withAccounting)
	require.NoError(t, err)
	assert.True(t, held.Suspended())
	assert.Equal(
		t,
		[]platformcatalog.FeatureKey{"tms.accounting"},
		held.Features,
		"the plan is kept",
	)

	fresh := organization()
	_, err = repo.SetLifecycle(ctx, &managedfeaturesports.LifecycleRequest{
		Scope:     fresh,
		Lifecycle: managedfeatures.LifecycleSuspended,
	})
	require.NoError(t, err)
	created, err := repo.Get(ctx, fresh)
	require.NoError(t, err)
	assert.False(t, created.HasList(), "a scope suspended before its first list has none")
}

func TestOnlyDueScopesAreFoundForTheReminder(t *testing.T) {
	t.Parallel()

	ctx, _, repo := newRepo(t)
	const (
		day = int64(86_400)
		now = int64(1_794_650_400)
	)
	set := func(scope managedscope.Scope, endsAt int64) {
		t.Helper()
		_, err := repo.Replace(ctx, &managedfeaturesports.ReplaceRequest{
			Scope:    scope,
			Received: &managedfeatures.Received{PlanEndsAt: endsAt},
			Now:      now - 30*day,
		})
		require.NoError(t, err)
	}
	due, later, suspended := organization(), organization(), organization()
	set(due, now+2*day)
	set(later, now+20*day)
	set(suspended, now+day)
	_, err := repo.SetLifecycle(ctx, &managedfeaturesports.LifecycleRequest{
		Scope:     suspended,
		Lifecycle: managedfeatures.LifecycleSuspended,
	})
	require.NoError(t, err)

	found, err := repo.DuePlanEndReminders(
		ctx,
		&managedfeaturesports.DueRemindersRequest{Now: now, Limit: 10},
	)
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, due, found[0].Scope)
	assert.Equal(t, now+2*day, found[0].EndsAt)

	_, err = repo.RemindPlanEnd(ctx, due, now)
	require.NoError(t, err)
	found, err = repo.DuePlanEndReminders(
		ctx,
		&managedfeaturesports.DueRemindersRequest{Now: now, Limit: 10},
	)
	require.NoError(t, err)
	assert.Empty(t, found, "a reminded end is not found again")
}

func TestDuePlanEndRemindersPageInPlanEndOrder(t *testing.T) {
	t.Parallel()

	ctx, _, repo := newRepo(t)
	const (
		day = int64(86_400)
		now = int64(1_794_650_400)
	)
	set := func(scope managedscope.Scope, endsAt int64) {
		t.Helper()
		_, err := repo.Replace(ctx, &managedfeaturesports.ReplaceRequest{
			Scope:    scope,
			Received: &managedfeatures.Received{PlanEndsAt: endsAt},
			Now:      now - 30*day,
		})
		require.NoError(t, err)
	}
	first, second, third := organization(), organization(), organization()
	set(first, now+day)
	set(second, now+2*day)
	set(third, now+2*day+1)

	page, err := repo.DuePlanEndReminders(
		ctx,
		&managedfeaturesports.DueRemindersRequest{Now: now, Limit: 2},
	)
	require.NoError(t, err)
	require.Len(t, page, 2)
	assert.Equal(t, first, page[0].Scope)
	assert.Equal(t, second, page[1].Scope)

	next, err := repo.DuePlanEndReminders(
		ctx,
		&managedfeaturesports.DueRemindersRequest{Now: now, Limit: 2, After: &page[1]},
	)
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.Equal(t, third, next[0].Scope)
}
