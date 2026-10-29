//go:build integration

package benefitrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paybenefit"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paybenefit/paybenefitdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode/paycodedb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedBenefitPlan(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	code string,
) *paybenefit.BenefitPlan {
	t.Helper()

	payCode := &paycode.PayCode{
		Direction: paycode.PayCodeDirectionDeduction,
		Code:      "PC" + code,
		Name:      code + " deduction",
	}
	require.NoError(t, paycodedb.PayCodes.Create(tenant.Ctx(ctx), db, payCode))

	entity := &paybenefit.BenefitPlan{
		Code:      code,
		Name:      code + " plan",
		PayCodeID: payCode.ID,
		PlanYear:  2026,
	}
	require.NoError(t, paybenefitdb.BenefitPlans.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetPlansByIDsReturnsOnlyTheTenantsPlans(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "BP")
	first := seedBenefitPlan(t, ctx, db, tenantA, "MEDICAL")
	second := seedBenefitPlan(t, ctx, db, tenantA, "DENTAL")
	foreign := seedBenefitPlan(t, ctx, db, tenantB, "VISION")

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	entities, err := repo.GetPlansByIDs(
		tenantA.Ctx(ctx),
		&driverpayports.GetBenefitPlansByIDsRequest{
			TenantInfo: tenantA.Info(),
			PlanIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("bplan_")},
		},
	)
	require.NoError(t, err)

	keys := make(map[pulid.ID]string, len(entities))
	for _, entity := range entities {
		keys[entity.ID] = entity.Code
		assert.Nil(t, entity.PayCode, "a batch read loads no pay code")
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "MEDICAL", second.ID: "DENTAL"}, keys)

	none, err := repo.GetPlansByIDs(
		tenantA.Ctx(ctx),
		&driverpayports.GetBenefitPlansByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}
