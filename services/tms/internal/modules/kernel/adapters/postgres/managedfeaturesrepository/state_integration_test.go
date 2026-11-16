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
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
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

func replace(
	t *testing.T,
	ctx context.Context,
	repo managedfeaturesports.StateRepository,
	now int64,
	features ...platformcatalog.FeatureKey,
) *managedfeatures.Replacement {
	t.Helper()
	replacement, err := repo.Replace(ctx, &managedfeaturesports.ReplaceRequest{
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

	held, changes, err := repo.History(ctx, 20)
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

	err := dbtx.WriteErr(dbscope.WithSystem(ctx, "test the managed feature list write"), postgres.WrapDB(db),
		func(ctx context.Context) error {
			if _, replaceErr := repo.Replace(ctx, &managedfeaturesports.ReplaceRequest{
				Received: &managedfeatures.Received{
					Features: []platformcatalog.FeatureKey{"ai.desk"},
				},
				Now: 100,
			}); replaceErr != nil {
				return replaceErr
			}
			return rollback
		})
	require.ErrorIs(t, err, rollback)

	held, changes, err := repo.History(ctx, 20)
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

	_, changes, err := repo.History(ctx, managedfeatures.ChangeHistoryLimit+50)
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

	held, changes, err := repo.History(ctx, 20)
	require.NoError(t, err)
	assert.Equal(t, planURL, held.PlanURL)
	assert.Equal(t, endings, held.AddOnEndings)
	require.Len(t, changes, 2)
	assert.Equal(t, endings, changes[0].Ending)
	assert.Empty(t, changes[0].Added)

	called := []managedfeatures.AddOnEnding{}
	kept, err := repo.Replace(ctx, &managedfeaturesports.ReplaceRequest{
		Received: &managedfeatures.Received{
			Features:     []platformcatalog.FeatureKey{"tms.accounting"},
			AddOnEndings: &called,
		},
		Now: 400,
	})
	require.NoError(t, err)
	require.True(t, kept.Changed())

	held, changes, err = repo.History(ctx, 20)
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
		replacement, err := repo.RemindPlanEnd(ctx, at)
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

	held, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, end, held.PlanEndsAt)
	assert.Equal(t, end, held.PlanEndAnnounced)
	assert.Equal(t, end, held.PlanEndReminded)

	kept := ending(0, end-day+60)
	require.NotNil(t, kept.PlanEnd)
	assert.Equal(t, managedfeatures.PlanEndKept, kept.PlanEnd.Kind)

	held, err = repo.Get(ctx)
	require.NoError(t, err)
	assert.Zero(t, held.PlanEndsAt)
	assert.Zero(t, held.PlanEndAnnounced)
	assert.Zero(t, held.PlanEndReminded)
}
