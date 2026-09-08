//go:build integration

package repositories

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupAccessPolicyCacheRepository(t *testing.T) (*accessPolicyCacheRepository, func()) {
	t.Helper()

	client := testutil.SetupTestRedis(t)
	repo := &accessPolicyCacheRepository{
		client: client,
		l:      zap.NewNop().Named("test.access-policy-cache"),
	}

	return repo, func() { client.FlushDB(t.Context()) }
}

func TestAccessPolicyCacheRepository_RoundTrip_Integration(t *testing.T) {
	repo, cleanup := setupAccessPolicyCacheRepository(t)
	defer cleanup()

	ctx := t.Context()
	lookup := repositories.IAMTenantPolicyLookupRequest{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	_, found, err := repo.GetEnabled(ctx, lookup)
	require.NoError(t, err)
	assert.False(t, found)

	policies := []*iam.AccessPolicy{
		{
			ID:             pulid.MustNew("ap_"),
			OrganizationID: lookup.OrganizationID,
			BusinessUnitID: lookup.BusinessUnitID,
			Name:           "Deny shipment read",
			Resource:       "shipment",
			Operation:      "read",
			Effect:         iam.PolicyEffectDeny,
			Priority:       10,
			Conditions:     map[string]string{"riskDecision": "step_up"},
			Enabled:        true,
		},
	}
	require.NoError(t, repo.SetEnabled(ctx, lookup, policies))

	cached, found, err := repo.GetEnabled(ctx, lookup)
	require.NoError(t, err)
	require.True(t, found)
	require.Len(t, cached, 1)
	assert.Equal(t, policies[0].ID, cached[0].ID)
	assert.Equal(t, iam.PolicyEffectDeny, cached[0].Effect)
	assert.Equal(t, "step_up", cached[0].Conditions["riskDecision"])

	require.NoError(t, repo.Invalidate(ctx, lookup))

	_, found, err = repo.GetEnabled(ctx, lookup)
	require.NoError(t, err)
	assert.False(t, found)
}

func TestAccessPolicyCacheRepository_CachesEmptyList_Integration(t *testing.T) {
	repo, cleanup := setupAccessPolicyCacheRepository(t)
	defer cleanup()

	ctx := t.Context()
	lookup := repositories.IAMTenantPolicyLookupRequest{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	require.NoError(t, repo.SetEnabled(ctx, lookup, nil))

	cached, found, err := repo.GetEnabled(ctx, lookup)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Empty(t, cached)
}

func TestAccessPolicyCacheRepository_TenantsAreIsolated_Integration(t *testing.T) {
	repo, cleanup := setupAccessPolicyCacheRepository(t)
	defer cleanup()

	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	first := repositories.IAMTenantPolicyLookupRequest{
		OrganizationID: orgID,
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	second := repositories.IAMTenantPolicyLookupRequest{
		OrganizationID: orgID,
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	require.NoError(t, repo.SetEnabled(ctx, first, []*iam.AccessPolicy{
		{ID: pulid.MustNew("ap_"), Resource: "shipment", Operation: "read"},
	}))

	_, found, err := repo.GetEnabled(ctx, second)
	require.NoError(t, err)
	assert.False(t, found)

	require.NoError(t, repo.Invalidate(ctx, second))

	cached, found, err := repo.GetEnabled(ctx, first)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Len(t, cached, 1)
}
