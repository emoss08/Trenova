//go:build integration

package accessorialchargerepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedAccessorialCharge(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	code string,
) *accessorialcharge.AccessorialCharge {
	t.Helper()

	entity := &accessorialcharge.AccessorialCharge{
		Code:        code,
		Description: code + " charge",
		Method:      accessorialcharge.MethodFlat,
		Amount:      decimal.NewFromInt(75),
	}
	require.NoError(t, accessorialchargedb.AccessorialCharges.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetByIDsReturnsOnlyTheTenantsAccessorialCharges(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "AC")
	first := seedAccessorialCharge(t, ctx, db, tenantA, "LUMPER")
	second := seedAccessorialCharge(t, ctx, db, tenantA, "DETENTION")
	foreign := seedAccessorialCharge(t, ctx, db, tenantB, "STOPOFF")

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	entities, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		accessorialchargeports.GetAccessorialChargesByIDsRequest{
			TenantInfo: tenantA.Info(),
			AccessorialChargeIDs: []pulid.ID{
				second.ID,
				foreign.ID,
				first.ID,
				pulid.MustNew("acc_"),
			},
		},
	)
	require.NoError(t, err)

	codes := make(map[pulid.ID]string, len(entities))
	for _, entity := range entities {
		codes[entity.ID] = entity.Code
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "LUMPER", second.ID: "DETENTION"}, codes)

	none, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		accessorialchargeports.GetAccessorialChargesByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}
