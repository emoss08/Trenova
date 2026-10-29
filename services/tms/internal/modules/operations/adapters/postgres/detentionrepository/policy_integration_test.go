//go:build integration

package detentionrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy/detentionpolicydb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedDetentionPolicies(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	codes ...string,
) []*detentionpolicy.DetentionPolicy {
	t.Helper()

	scoped := tenant.Ctx(ctx)
	charge := &accessorialcharge.AccessorialCharge{
		Code:        "DETENTION",
		Description: "Detention",
		Method:      accessorialcharge.MethodFlat,
		Amount:      decimal.NewFromInt(75),
	}
	require.NoError(t, accessorialchargedb.AccessorialCharges.Create(scoped, db, charge))

	policies := make([]*detentionpolicy.DetentionPolicy, 0, len(codes))
	for _, code := range codes {
		policy := &detentionpolicy.DetentionPolicy{
			Name:                code + " policy",
			Code:                code,
			AccessorialChargeID: charge.ID,
		}
		require.NoError(t, detentionpolicydb.DetentionPolicies.Create(scoped, db, policy))
		policies = append(policies, policy)
	}

	return policies
}

func TestGetByIDsReturnsOnlyTheTenantsPolicies(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "DT")
	own := seedDetentionPolicies(t, ctx, db, tenantA, "SHIPPER", "RECEIVER")
	foreign := seedDetentionPolicies(t, ctx, db, tenantB, "FOREIGN")[0]

	repo := NewPolicyRepository(PolicyParams{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	policies, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		&detentionports.GetDetentionPoliciesByIDsRequest{
			TenantInfo: tenantA.Info(),
			DetentionPolicyIDs: []pulid.ID{
				own[1].ID,
				foreign.ID,
				own[0].ID,
				pulid.MustNew("dtp_"),
			},
		},
	)
	require.NoError(t, err)

	codes := make(map[pulid.ID]string, len(policies))
	for _, policy := range policies {
		codes[policy.ID] = policy.Code
		assert.Empty(t, policy.Tiers, "a batch read loads no tiers")
	}
	assert.Equal(t, map[pulid.ID]string{own[0].ID: "SHIPPER", own[1].ID: "RECEIVER"}, codes)

	none, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		&detentionports.GetDetentionPoliciesByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}
