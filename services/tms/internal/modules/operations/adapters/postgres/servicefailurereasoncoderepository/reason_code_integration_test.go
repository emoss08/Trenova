//go:build integration

package servicefailurereasoncoderepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedReasonCode(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	code string,
) *servicefailure.ReasonCode {
	t.Helper()

	entity := &servicefailure.ReasonCode{
		Code:      code,
		Label:     "Reason " + code,
		Category:  servicefailure.ReasonCategoryWeather,
		AppliesTo: servicefailure.ReasonCodeAppliesToBoth,
		Active:    true,
		SortOrder: 10,
	}
	require.NoError(t, servicefailuredb.ReasonCodes.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetByIDsReturnsOnlyTheTenantsReasonCodes(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "RC")
	first := seedReasonCode(t, ctx, db, tenantA, "WX1")
	second := seedReasonCode(t, ctx, db, tenantA, "WX2")
	foreign := seedReasonCode(t, ctx, db, tenantB, "WX3")

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	entities, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		servicefailureports.GetServiceFailureReasonCodesByIDsRequest{
			TenantInfo:    tenantA.Info(),
			ReasonCodeIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("sfrc_")},
		},
	)
	require.NoError(t, err)

	codes := make(map[pulid.ID]string, len(entities))
	for _, entity := range entities {
		codes[entity.ID] = entity.Code
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "WX1", second.ID: "WX2"}, codes)
}
