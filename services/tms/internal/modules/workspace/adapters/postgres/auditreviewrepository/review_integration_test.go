//go:build integration

package auditreviewrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/auditreviewrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const occurred = int64(1_800_000_000)

type harness struct {
	ctx      context.Context
	repo     auditports.ReviewRepository
	tenant   pagination.TenantInfo
	userID   pulid.ID
	other    pagination.TenantInfo
	otherCtx context.Context
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "AR")
	tenant := data.Info()
	return &harness{
		ctx: dbscope.WithTenant(ctx, tenant.DBTenant()),
		repo: auditreviewrepository.New(
			auditreviewrepository.Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()},
		),
		tenant:   tenant,
		userID:   data.User.ID,
		other:    other.Info(),
		otherCtx: other.Ctx(ctx),
	}
}

func (h *harness) review(dedupe string, entryID pulid.ID, at int64) *audit.Review {
	return &audit.Review{
		Kind:          audit.AlertBulkDelete,
		DedupeKey:     dedupe,
		EntryID:       entryID,
		Action:        "accessorial_rate.delete",
		Resource:      permission.Resource("accessorial_rate"),
		PrincipalID:   pulid.ID("usr_dana"),
		PrincipalType: "session_user",
		ActorName:     "Dana Whitfield",
		Count:         50,
		Threshold:     50,
		WindowMinutes: 10,
		OccurredAt:    at,
	}
}

func TestRaise_ARepeatedAlertFindsTheReviewItAlreadyRaised(t *testing.T) {
	h := newHarness(t)

	first, created, err := h.repo.Raise(h.ctx, h.review("BulkDelete/usr_dana:1", "ae_1", occurred))
	require.NoError(t, err)
	assert.True(t, created)

	again, created, err := h.repo.Raise(h.ctx, h.review("BulkDelete/usr_dana:1", "ae_1", occurred))
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, first.ID, again.ID)
}

func TestList_OpenLatestFirstAndAWeekOfReviewed(t *testing.T) {
	h := newHarness(t)

	older, _, err := h.repo.Raise(h.ctx, h.review("a", "ae_1", occurred-60))
	require.NoError(t, err)
	newer, _, err := h.repo.Raise(h.ctx, h.review("b", "ae_2", occurred))
	require.NoError(t, err)
	done, _, err := h.repo.Raise(h.ctx, h.review("c", "ae_3", occurred-120))
	require.NoError(t, err)
	_, err = h.repo.SetReviewed(h.ctx, &auditports.SetReviewedRequest{
		TenantInfo: h.tenant, ID: done.ID, ReviewerID: h.userID, At: occurred + 10, Reviewed: true,
	})
	require.NoError(t, err)

	list, err := h.repo.List(h.ctx, &auditports.ListReviewsRequest{
		TenantInfo: h.tenant, Limit: 10, ReviewedSince: occurred,
	})
	require.NoError(t, err)

	require.Len(t, list.Open, 2)
	assert.Equal(t, newer.ID, list.Open[0].ID)
	assert.Equal(t, older.ID, list.Open[1].ID)
	assert.Equal(t, 2, list.OpenCount)
	require.Len(t, list.Reviewed, 1)
	assert.Equal(t, done.ID, list.Reviewed[0].ID)
	assert.Equal(t, 1, list.ReviewedCount)

	stale, err := h.repo.List(h.ctx, &auditports.ListReviewsRequest{
		TenantInfo: h.tenant, Limit: 10, ReviewedSince: occurred + 11,
	})
	require.NoError(t, err)
	assert.Empty(t, stale.Reviewed, "a review older than the window is not listed")
}

func TestSetReviewed_OnlyFromTheOtherState(t *testing.T) {
	h := newHarness(t)
	raised, _, err := h.repo.Raise(h.ctx, h.review("a", "ae_1", occurred))
	require.NoError(t, err)

	reviewed, err := h.repo.SetReviewed(h.ctx, &auditports.SetReviewedRequest{
		TenantInfo: h.tenant, ID: raised.ID, ReviewerID: h.userID, At: occurred + 5, Reviewed: true,
	})
	require.NoError(t, err)
	require.NotNil(t, reviewed.ReviewedAt)
	assert.Equal(t, occurred+5, *reviewed.ReviewedAt)
	assert.Equal(t, h.userID, reviewed.ReviewedByID)
	assert.Greater(t, reviewed.Version, raised.Version)

	_, err = h.repo.SetReviewed(h.ctx, &auditports.SetReviewedRequest{
		TenantInfo: h.tenant, ID: raised.ID, ReviewerID: h.userID, At: occurred + 6, Reviewed: true,
	})
	require.ErrorIs(
		t,
		err,
		auditports.ErrReviewUnchanged,
		"a second reviewer does not overwrite the first",
	)

	reopened, err := h.repo.SetReviewed(h.ctx, &auditports.SetReviewedRequest{
		TenantInfo: h.tenant, ID: raised.ID, Reviewed: false,
	})
	require.NoError(t, err)
	assert.Nil(t, reopened.ReviewedAt)
	assert.True(t, reopened.ReviewedByID.IsNil())
}

func TestGetByEntryIDs_FindsTheReviewAnEntrySetOff(t *testing.T) {
	h := newHarness(t)
	raised, _, err := h.repo.Raise(h.ctx, h.review("a", "ae_1", occurred))
	require.NoError(t, err)

	found, err := h.repo.GetByEntryIDs(h.ctx, h.tenant, []pulid.ID{"ae_1", "ae_none"})
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, raised.ID, found[0].ID)
}

func TestAnotherTenantReadsNoneOfTheReviews(t *testing.T) {
	h := newHarness(t)
	raised, _, err := h.repo.Raise(h.ctx, h.review("a", "ae_1", occurred))
	require.NoError(t, err)

	list, err := h.repo.List(
		h.otherCtx,
		&auditports.ListReviewsRequest{TenantInfo: h.other, Limit: 10},
	)
	require.NoError(t, err)
	assert.Empty(t, list.Open)
	assert.Zero(t, list.OpenCount)

	_, err = h.repo.GetByID(h.otherCtx, h.other, raised.ID)
	require.Error(t, err)

	_, err = h.repo.SetReviewed(h.otherCtx, &auditports.SetReviewedRequest{
		TenantInfo: h.other, ID: raised.ID, Reviewed: true, At: occurred,
	})
	require.ErrorIs(t, err, auditports.ErrReviewUnchanged)
}
