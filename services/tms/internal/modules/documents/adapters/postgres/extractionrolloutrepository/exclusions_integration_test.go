//go:build integration

package extractionrolloutrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionrolloutports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestRolloutExclusionsAgainstPostgres records exclusions against the real
// schema: one row per extraction however often it is recorded, totals per
// reason since the rollout started for its candidate, and the retention purge.
func TestRolloutExclusionsAgainstPostgres(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := NewExclusions(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	ti := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	candidate := pulid.MustNew("aip_")
	other := pulid.MustNew("aip_")

	exclusion := func(document pulid.ID, provider pulid.ID, createdAt int64) *extractionrollout.RolloutExclusion {
		return &extractionrollout.RolloutExclusion{
			OrganizationID:      ti.OrgID,
			BusinessUnitID:      ti.BuID,
			DocumentID:          document,
			ExtractedAt:         1_800_000_000,
			CandidateProviderID: provider,
			Reason:              extractionrollout.ExclusionReasonDoclingInput,
			CreatedAt:           createdAt,
		}
	}

	retried := pulid.MustNew("doc_")
	require.NoError(t, repo.Record(ctx, exclusion(retried, candidate, 2_000)))
	require.NoError(t, repo.Record(ctx, exclusion(retried, candidate, 2_000)))
	require.NoError(t, repo.Record(ctx, exclusion(pulid.MustNew("doc_"), candidate, 2_500)))
	require.NoError(t, repo.Record(ctx, exclusion(pulid.MustNew("doc_"), candidate, 500)))
	require.NoError(t, repo.Record(ctx, exclusion(pulid.MustNew("doc_"), other, 2_000)))

	totals, err := repo.Totals(ctx, extractionrolloutports.TotalRolloutAssignmentsRequest{
		TenantInfo:          ti,
		CandidateProviderID: candidate,
		Since:               1_000,
	})
	require.NoError(t, err)
	assert.Equal(t, []extractionrolloutports.RolloutExclusionTotal{{
		Reason: extractionrollout.ExclusionReasonDoclingInput,
		Count:  2,
	}}, totals)

	purged, err := repo.PurgeBefore(ctx, extractionrolloutports.PurgeRolloutAssignmentsRequest{
		TenantInfo: ti,
		Before:     1_000,
		Limit:      10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), purged)
}
