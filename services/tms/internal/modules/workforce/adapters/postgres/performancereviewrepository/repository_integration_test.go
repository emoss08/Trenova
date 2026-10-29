//go:build integration

package performancereviewrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview/workerreviewdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerreviewports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedReviewTemplate(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	code string,
) *workerreview.PerformanceReviewTemplate {
	t.Helper()

	entity := &workerreview.PerformanceReviewTemplate{Code: code, Name: code + " review"}
	require.NoError(t, workerreviewdb.PerformanceReviewTemplates.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetTemplatesByIDsReturnsOnlyTheTenantsTemplates(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "RT")
	first := seedReviewTemplate(t, ctx, db, tenantA, "ANNUAL")
	second := seedReviewTemplate(t, ctx, db, tenantA, "PROBATION")
	foreign := seedReviewTemplate(t, ctx, db, tenantB, "QUARTERLY")

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	entities, err := repo.GetTemplatesByIDs(
		tenantA.Ctx(ctx),
		&workerreviewports.GetReviewTemplatesByIDsRequest{
			TenantInfo: tenantA.Info(),
			TemplateIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("prt_")},
		},
	)
	require.NoError(t, err)

	keys := make(map[pulid.ID]string, len(entities))
	for _, entity := range entities {
		keys[entity.ID] = entity.Code
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "ANNUAL", second.ID: "PROBATION"}, keys)

	none, err := repo.GetTemplatesByIDs(
		tenantA.Ctx(ctx),
		&workerreviewports.GetReviewTemplatesByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}
