//go:build integration

package ptopolicyrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto/ptodb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/ptopolicyports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedPTOPolicy(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	code string,
) *pto.PTOPolicy {
	t.Helper()

	entity := &pto.PTOPolicy{Code: code, Name: code + " policy"}
	require.NoError(t, ptodb.PTOPolicies.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetByIDsReturnsOnlyTheTenantsPoliciesWithTheirRules(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "PT")
	first := seedPTOPolicy(t, ctx, db, tenantA, "STANDARD")
	second := seedPTOPolicy(t, ctx, db, tenantA, "SENIOR")
	foreign := seedPTOPolicy(t, ctx, db, tenantB, "FOREIGN")

	rule := &pto.PTOPolicyRule{PTOPolicyID: first.ID, PTOType: worker.PTOTypeVacation}
	require.NoError(t, ptodb.PTOPolicyRules.Create(tenantA.Ctx(ctx), db, rule))

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	entities, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		&ptopolicyports.GetPTOPoliciesByIDsRequest{
			TenantInfo: tenantA.Info(),
			PolicyIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("ptop_")},
			IncludeRules: true,
		},
	)
	require.NoError(t, err)

	keys := make(map[pulid.ID]string, len(entities))
	for _, entity := range entities {
		keys[entity.ID] = entity.Code
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "STANDARD", second.ID: "SENIOR"}, keys)

	for _, entity := range entities {
		if entity.ID == first.ID {
			require.Len(t, entity.Rules, 1)
			assert.Equal(t, worker.PTOTypeVacation, entity.Rules[0].PTOType)
		} else {
			assert.Empty(t, entity.Rules)
		}
	}

	none, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		&ptopolicyports.GetPTOPoliciesByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}
