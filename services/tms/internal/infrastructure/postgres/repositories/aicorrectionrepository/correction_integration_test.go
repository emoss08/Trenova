//go:build integration

package aicorrectionrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type tenantRow struct {
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func TestProviderTotals_BucketCorrectionsByProviderAndWeek(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	cols := buncolgen.DefinitionColumns
	var row tenantRow
	require.NoError(t, db.NewSelect().
		TableExpr(buncolgen.DefinitionTable.As(buncolgen.DefinitionTable.Alias)).
		Column(cols.OrganizationID.Bare(), cols.BusinessUnitID.Bare()).
		Limit(1).
		Scan(ctx, &row))
	tenant := pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID}

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
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

	weekly, err := repo.WeeklyTotalsByProvider(ctx, &repositories.WeeklyAICorrectionTotalsRequest{
		TenantInfo: tenant,
		Task:       aicorrection.TaskShipmentDraftExtraction,
		Since:      monday,
	})
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

	split, err := repo.TotalsByProvider(ctx, &repositories.TotalAICorrectionsByProviderRequest{
		TenantInfo: tenant,
		Task:       aicorrection.TaskShipmentDraftExtraction,
		ProviderID: candidate,
		Since:      monday,
	})
	require.NoError(t, err)
	sides := make(map[bool]repositories.AICorrectionProviderTotal, len(split))
	for _, side := range split {
		sides[side.Candidate] = side
	}
	assert.Equal(t, 35, sides[true].Scored)
	assert.Equal(t, 29, sides[true].Correct)
	assert.Equal(t, 40, sides[false].Scored)
}
