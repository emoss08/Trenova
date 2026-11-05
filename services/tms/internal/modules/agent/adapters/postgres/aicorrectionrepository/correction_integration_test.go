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
	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
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
}
