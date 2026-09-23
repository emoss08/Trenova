//go:build integration

package aifeedbackrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func feedbackFor(
	tenant pagination.TenantInfo,
	userID pulid.ID,
	target repositories.AIFeedbackTargetRef,
	rating aifeedback.Rating,
) *aifeedback.Feedback {
	agentID := pulid.MustNew("agdef_")
	reasons := []aifeedback.Reason{aifeedback.ReasonHelpful}
	if rating == aifeedback.RatingNegative {
		reasons = []aifeedback.Reason{aifeedback.ReasonInaccurate}
	}

	return &aifeedback.Feedback{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		UserID:            userID,
		TargetType:        target.TargetType,
		TargetID:          target.TargetID,
		TargetPart:        target.TargetPart,
		AgentDefinitionID: &agentID,
		FingerprintSource: aifeedback.FingerprintAtRating,
		Rating:            rating,
		Reasons:           reasons,
		Comment:           "The lane was wrong",
		TurnSnapshot: aifeedback.NewTurnSnapshot(aifeedback.SnapshotParams{
			Question: "Where is load 12?",
			Answer:   "In Dallas.",
		}),
		PatternKey: aifeedback.PatternKey([]string{"get_shipment"}, reasons[0], "Shipment"),
	}
}

func TestFeedbackRepository_UpsertClearAndTenantIsolation(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	first := seedtest.SeedFullTestData(t, ctx, db)
	second := seedtest.SeedAdditionalTenant(t, ctx, db, "FK")

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	firstTenant := pagination.TenantInfo{OrgID: first.Organization.ID, BuID: first.BusinessUnit.ID}
	secondTenant := pagination.TenantInfo{
		OrgID: second.Organization.ID,
		BuID:  second.BusinessUnit.ID,
	}
	target := repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetAssistantMessage,
		TargetID:   pulid.MustNew("amsg_"),
	}

	created, err := repo.Upsert(ctx, feedbackFor(firstTenant, first.User.ID, target,
		aifeedback.RatingPositive))
	require.NoError(t, err)
	require.False(t, created.ID.IsNil())

	changed, err := repo.Upsert(ctx, feedbackFor(firstTenant, first.User.ID, target,
		aifeedback.RatingNegative))
	require.NoError(t, err)
	assert.Equal(t, created.ID, changed.ID, "rating again replaces the row")
	assert.Equal(t, aifeedback.RatingNegative, changed.Rating)
	assert.Equal(t, int64(1), changed.Version)

	_, err = repo.Upsert(ctx, feedbackFor(secondTenant, second.User.ID, target,
		aifeedback.RatingPositive))
	require.NoError(t, err)

	mine, err := repo.ListForTargets(ctx, repositories.ListAIFeedbackForTargetsRequest{
		TenantInfo: firstTenant,
		UserID:     first.User.ID,
		Targets:    []repositories.AIFeedbackTargetRef{target},
	})
	require.NoError(t, err)
	require.Len(t, mine, 1)
	assert.Equal(t, aifeedback.RatingNegative, mine[0].Rating)
	assert.Equal(t, []aifeedback.Reason{aifeedback.ReasonInaccurate}, mine[0].Reasons)
	require.NotNil(t, mine[0].TurnSnapshot)
	assert.Equal(t, "In Dallas.", mine[0].TurnSnapshot.Answer)

	crossed, err := repo.ListForTargets(ctx, repositories.ListAIFeedbackForTargetsRequest{
		TenantInfo: secondTenant,
		UserID:     first.User.ID,
		Targets:    []repositories.AIFeedbackTargetRef{target},
	})
	require.NoError(t, err)
	assert.Empty(t, crossed, "a rating is never read from another tenant")

	byIDs, err := repo.ListByIDs(ctx, repositories.ListAIFeedbackByIDsRequest{
		TenantInfo: secondTenant,
		IDs:        []pulid.ID{created.ID},
	})
	require.NoError(t, err)
	assert.Empty(t, byIDs, "an id from another tenant reads nothing")

	removed, err := repo.Delete(ctx, repositories.DeleteAIFeedbackRequest{
		TenantInfo: secondTenant,
		UserID:     first.User.ID,
		Target:     target,
	})
	require.NoError(t, err)
	assert.False(t, removed, "clearing in another tenant touches nothing")

	removed, err = repo.Delete(ctx, repositories.DeleteAIFeedbackRequest{
		TenantInfo: firstTenant,
		UserID:     first.User.ID,
		Target:     target,
	})
	require.NoError(t, err)
	assert.True(t, removed)

	removed, err = repo.Delete(ctx, repositories.DeleteAIFeedbackRequest{
		TenantInfo: firstTenant,
		UserID:     first.User.ID,
		Target:     target,
	})
	require.NoError(t, err)
	assert.False(t, removed, "clearing twice is not an error")

	theirs, err := repo.ListForTargets(ctx, repositories.ListAIFeedbackForTargetsRequest{
		TenantInfo: secondTenant,
		UserID:     second.User.ID,
		Targets:    []repositories.AIFeedbackTargetRef{target},
	})
	require.NoError(t, err)
	assert.Len(t, theirs, 1, "the other tenant's rating survives")
}

func TestFeedbackRepository_SummaryNegativeScanAndPurge(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "FS")
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	otherTenant := pagination.TenantInfo{OrgID: other.Organization.ID, BuID: other.BusinessUnit.ID}

	agentID := pulid.MustNew("agdef_")
	worst := repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetAssistantMessage,
		TargetID:   pulid.MustNew("amsg_"),
	}
	liked := repositories.AIFeedbackTargetRef{
		TargetType: aifeedback.TargetAssistantMessage,
		TargetID:   pulid.MustNew("amsg_"),
	}

	for _, entry := range []struct {
		tenant pagination.TenantInfo
		user   pulid.ID
		target repositories.AIFeedbackTargetRef
		rating aifeedback.Rating
	}{
		{tenant, data.User.ID, worst, aifeedback.RatingNegative},
		{tenant, data.User.ID, liked, aifeedback.RatingPositive},
		{otherTenant, other.User.ID, worst, aifeedback.RatingNegative},
	} {
		feedback := feedbackFor(entry.tenant, entry.user, entry.target, entry.rating)
		feedback.AgentDefinitionID = &agentID
		_, err := repo.Upsert(ctx, feedback)
		require.NoError(t, err)
	}

	window := repositories.AIFeedbackWindowRequest{
		TenantInfo:        tenant,
		AgentDefinitionID: agentID,
		Since:             timeutils.NowUnix() - 86400,
		Timezone:          "America/Chicago",
		Limit:             5,
	}

	days, err := repo.DailySatisfaction(ctx, window)
	require.NoError(t, err)
	require.Len(t, days, 1)
	assert.Equal(t, 1, days[0].Positive)
	assert.Equal(t, 1, days[0].Negative, "the other tenant's thumbs down is not counted")

	scores, err := repo.WorstRated(ctx, window)
	require.NoError(t, err)
	require.Len(t, scores, 1, "only targets with a thumbs down are listed")
	assert.Equal(t, worst.TargetID, scores[0].TargetID)
	assert.Equal(t, 1, scores[0].Negative)
	assert.False(t, scores[0].SampleID.IsNil())

	negatives, err := repo.ListNegativeSince(ctx, repositories.ListNegativeAIFeedbackRequest{
		TenantInfo: tenant,
		Since:      window.Since,
	})
	require.NoError(t, err)
	require.Len(t, negatives, 1)
	assert.Equal(t, worst.TargetID, negatives[0].TargetID)

	purged, err := repo.PurgeBefore(ctx, repositories.PurgeAIFeedbackRequest{
		TenantInfo: tenant,
		Before:     timeutils.NowUnix() + 60,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), purged)

	survivors, err := repo.ListNegativeSince(ctx, repositories.ListNegativeAIFeedbackRequest{
		TenantInfo: otherTenant,
		Since:      window.Since,
	})
	require.NoError(t, err)
	assert.Len(t, survivors, 1, "purging one tenant leaves the other's ratings")
}
