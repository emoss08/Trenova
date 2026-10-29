//go:build integration

package schedulingrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule/workerscheduledb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerscheduleports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedShiftTemplate(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	code string,
) *workerschedule.ShiftTemplate {
	t.Helper()

	entity := &workerschedule.ShiftTemplate{Code: code, Name: code + " shift"}
	require.NoError(t, workerscheduledb.ShiftTemplates.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetTemplatesByIDsReturnsOnlyTheTenantsTemplates(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "ST")
	first := seedShiftTemplate(t, ctx, db, tenantA, "DAYS")
	second := seedShiftTemplate(t, ctx, db, tenantA, "NIGHTS")
	foreign := seedShiftTemplate(t, ctx, db, tenantB, "SWING")

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	entities, err := repo.GetTemplatesByIDs(
		tenantA.Ctx(ctx),
		&workerscheduleports.GetShiftTemplatesByIDsRequest{
			TenantInfo: tenantA.Info(),
			TemplateIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("shft_")},
		},
	)
	require.NoError(t, err)

	keys := make(map[pulid.ID]string, len(entities))
	for _, entity := range entities {
		keys[entity.ID] = entity.Code
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "DAYS", second.ID: "NIGHTS"}, keys)

	none, err := repo.GetTemplatesByIDs(
		tenantA.Ctx(ctx),
		&workerscheduleports.GetShiftTemplatesByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}
