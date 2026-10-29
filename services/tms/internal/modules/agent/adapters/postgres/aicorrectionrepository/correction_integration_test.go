//go:build integration

package aicorrectionrepository

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol/agentcontroldb"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type tenantRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func seededTenant(t *testing.T) (context.Context, *keel.DB, pagination.TenantInfo) {
	t.Helper()

	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	d := agentdefinitiondb.Definitions
	row, err := keel.IntoOne[tenantRow](
		dbscope.WithSystem(ctx, "test"),
		db,
		d.AllTenants().Pick(d.OrganizationID, d.BusinessUnitID).Limit(1),
	)
	require.NoError(t, err)

	info := pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID}

	return dbscope.WithTenant(ctx, info.DBTenant()), db, info
}

func TestProviderTotals_BucketCorrectionsByProviderAndWeek(t *testing.T) {
	ctx, db, tenant := seededTenant(t)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	var err error
	candidate := pulid.MustNew("aip_")
	production := pulid.MustNew("aip_")
	monday := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC).Unix()
	sunday := time.Date(2026, time.September, 27, 23, 59, 59, 0, time.UTC).Unix()
	nextMonday := time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC).Unix()

	for _, c := range []struct {
		provider   *pulid.ID
		capturedAt int64
		scored     int
		correct    int
	}{
		{provider: &candidate, capturedAt: monday, scored: 10, correct: 9},
		{provider: &candidate, capturedAt: sunday, scored: 20, correct: 15},
		{provider: &candidate, capturedAt: nextMonday, scored: 5, correct: 5},
		{provider: &production, capturedAt: sunday, scored: 40, correct: 30},
		{capturedAt: sunday, scored: 100, correct: 0},
	} {
		_, err = repo.Upsert(ctx, &aicorrection.Correction{
			OrganizationID:       tenant.OrgID,
			BusinessUnitID:       tenant.BuID,
			Task:                 aicorrection.TaskShipmentDraftExtraction,
			SourceType:           aicorrection.SourceDocumentShipmentDraft,
			SourceID:             pulid.MustNew("dsd_"),
			SubjectType:          aicorrection.SubjectShipment,
			SubjectID:            pulid.MustNew("shp_"),
			CapturedByID:         pulid.MustNew("usr_"),
			ExtractionProviderID: c.provider,
			Predicted:            &aicorrection.Snapshot{Fields: map[string]string{}},
			Confirmed:            &aicorrection.Snapshot{Fields: map[string]string{}},
			FieldResults:         []aicorrection.FieldResult{},
			ScoredCount:          c.scored,
			CorrectCount:         c.correct,
			CorrectedCount:       c.scored - c.correct,
			CapturedAt:           c.capturedAt,
		})
		require.NoError(t, err)
	}

	weekly, err := repo.WeeklyTotalsByProvider(
		ctx,
		&aicorrectionports.WeeklyAICorrectionTotalsRequest{
			TenantInfo: tenant,
			Task:       aicorrection.TaskShipmentDraftExtraction,
			Since:      monday,
		},
	)
	require.NoError(t, err)

	type key struct {
		provider pulid.ID
		week     int64
	}
	byKey := make(map[key]aicorrection.WeekTotal, len(weekly))
	for _, total := range weekly {
		byKey[key{total.ProviderID, total.WeekStart}] = total
	}
	assert.Len(t, weekly, 3, "a draft with no provider is not counted for any")
	assert.Equal(t, aicorrection.WeekTotal{
		ProviderID: candidate, WeekStart: monday, Corrections: 2, Scored: 30, Correct: 24,
	}, byKey[key{candidate, monday}], "Monday through Sunday is one UTC week")
	assert.Equal(t, 5, byKey[key{candidate, nextMonday}].Scored)
	assert.Equal(t, 40, byKey[key{production, monday}].Scored)

	split, err := repo.TotalsByProvider(ctx, &aicorrectionports.TotalAICorrectionsByProviderRequest{
		TenantInfo: tenant,
		Task:       aicorrection.TaskShipmentDraftExtraction,
		ProviderID: candidate,
		Since:      monday,
	})
	require.NoError(t, err)
	sides := make(map[bool]aicorrectionports.AICorrectionProviderTotal, len(split))
	for _, side := range split {
		sides[side.Candidate] = side
	}
	assert.Equal(t, 35, sides[true].Scored)
	assert.Equal(t, 29, sides[true].Correct)
	assert.Equal(t, 40, sides[false].Scored)
}

func TestTrainableCorrections_CountAndWeeklyTotalsFollowConsent(t *testing.T) {
	ctx, db, tenantInfo := seededTenant(t)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	provider := pulid.MustNew("aip_")
	from := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC).Unix()
	to := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC).Unix()
	monday := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC).Unix()
	document := func() *pulid.ID {
		id := pulid.MustNew("doc_")
		return &id
	}

	for _, c := range []struct {
		document   *pulid.ID
		provider   *pulid.ID
		capturedAt int64
		scored     int
	}{
		{document: document(), provider: &provider, capturedAt: monday, scored: 5},
		{document: document(), provider: &provider, capturedAt: monday + 3600, scored: 7},
		{document: document(), capturedAt: from, scored: 2},
		{capturedAt: monday, scored: 9},
		{document: document(), capturedAt: monday, scored: 0},
		{document: document(), capturedAt: from - 1, scored: 4},
		{document: document(), capturedAt: to, scored: 4},
	} {
		_, err := repo.Upsert(ctx, &aicorrection.Correction{
			OrganizationID:       tenantInfo.OrgID,
			BusinessUnitID:       tenantInfo.BuID,
			Task:                 aicorrection.TaskShipmentDraftExtraction,
			SourceType:           aicorrection.SourceDocumentShipmentDraft,
			SourceID:             pulid.MustNew("dsd_"),
			SubjectType:          aicorrection.SubjectShipment,
			SubjectID:            pulid.MustNew("shp_"),
			CapturedByID:         pulid.MustNew("usr_"),
			DocumentID:           c.document,
			ExtractionProviderID: c.provider,
			Predicted:            &aicorrection.Snapshot{Fields: map[string]string{}},
			Confirmed:            &aicorrection.Snapshot{Fields: map[string]string{}},
			FieldResults:         []aicorrection.FieldResult{},
			ScoredCount:          c.scored,
			CorrectCount:         c.scored,
			CapturedAt:           c.capturedAt,
		})
		require.NoError(t, err)
	}

	everyTenant := dbscope.WithSystem(ctx, "test")
	count := func(capPerOrganization int) int {
		t.Helper()
		total, err := repo.CountTrainable(
			everyTenant,
			&aicorrectionports.CountTrainableAICorrectionsRequest{
				Task:               aicorrection.TaskShipmentDraftExtraction,
				CapturedFrom:       from,
				CapturedTo:         to,
				PerOrganizationCap: capPerOrganization,
			},
		)
		require.NoError(t, err)
		return total
	}
	weekly := func() []aicorrection.WeekTotal {
		t.Helper()
		totals, err := repo.WeeklyTrainableTotalsByProvider(
			everyTenant,
			&aicorrectionports.WeeklyTrainableAICorrectionTotalsRequest{
				Task:  aicorrection.TaskShipmentDraftExtraction,
				Since: from,
			},
		)
		require.NoError(t, err)
		return totals
	}

	assert.Zero(t, count(100), "an organization without consent has nothing to train on")
	assert.Empty(t, weekly())

	controls := agentcontroldb.AgentControls
	updated, err := controls.Where(keel.True()).Update(ctx, db, controls.AITrainingConsent.To(true))
	require.NoError(t, err)
	if updated == 0 {
		require.NoError(t, controls.Create(ctx, db, &agentcontrol.AgentControl{
			OrganizationID:     tenantInfo.OrgID,
			BusinessUnitID:     tenantInfo.BuID,
			AITrainingConsent:  true,
			PromotionThreshold: agentcontrol.DefaultPromotionThreshold,
		}))
	}

	assert.Equal(t, 3, count(100),
		"only scored corrections with a document, inside the window, are trainable")

	first, err := repo.ListForTraining(ctx, &aicorrectionports.ListAICorrectionsForTrainingRequest{
		TenantInfo:   tenantInfo,
		Task:         aicorrection.TaskShipmentDraftExtraction,
		CapturedFrom: from,
		CapturedTo:   to,
		Limit:        2,
	})
	require.NoError(t, err)
	require.Len(t, first, 2)
	assert.Equal(t, monday+3600, first[0].CapturedAt, "the newest correction comes first")
	assert.Equal(t, monday, first[1].CapturedAt)

	rest, err := repo.ListForTraining(ctx, &aicorrectionports.ListAICorrectionsForTrainingRequest{
		TenantInfo:       tenantInfo,
		Task:             aicorrection.TaskShipmentDraftExtraction,
		CapturedFrom:     from,
		CapturedTo:       to,
		BeforeCapturedAt: first[1].CapturedAt,
		BeforeID:         first[1].ID,
		Limit:            2,
	})
	require.NoError(t, err)
	require.Len(t, rest, 1, "the cursor continues after the last correction read")
	assert.Equal(t, from, rest[0].CapturedAt)
	assert.Equal(t, 2, count(2), "an organization counts for at most its cap")

	totals := weekly()
	require.Len(t, totals, 1, "a draft with no provider is not counted for any")
	assert.Equal(t, aicorrection.WeekTotal{
		ProviderID: provider, WeekStart: monday, Corrections: 2, Scored: 12, Correct: 12,
	}, totals[0])

	_, err = repo.CountTrainable(everyTenant, &aicorrectionports.CountTrainableAICorrectionsRequest{
		Task:         aicorrection.TaskShipmentDraftExtraction,
		CapturedFrom: from,
		CapturedTo:   to,
	})
	require.Error(t, err, "a missing cap is refused rather than counting without one")
}
