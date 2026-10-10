//go:build integration

package airetrievalrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListIndexedTenants_ListsOnlyOrganizationsWithAnActiveModel(t *testing.T) {
	h := newHarness(t)
	list := func(req repositories.ListIndexedRetrievalTenantsRequest) int {
		t.Helper()
		tenants, err := h.repo.ListIndexedTenants(h.ctx, req)
		require.NoError(t, err)
		for _, tenant := range tenants {
			assert.Equal(t, h.tenant, tenant)
		}
		return len(tenants)
	}

	assert.Zero(t, list(repositories.ListIndexedRetrievalTenantsRequest{Limit: 10}),
		"an organization that never chose a model is not swept")

	settings := airetrieval.DefaultSettings(h.tenant.OrgID, h.tenant.BuID)
	settings.ActiveModelKey = "text-embedding-3-small"
	settings.Dimensions = 1536
	saved, err := h.repo.UpdateSettings(h.ctx, settings)
	require.NoError(t, err)

	assert.Equal(t, 1, list(repositories.ListIndexedRetrievalTenantsRequest{Limit: 10}))
	assert.Zero(t, list(repositories.ListIndexedRetrievalTenantsRequest{
		After: &h.tenant,
		Limit: 10,
	}), "the cursor is exclusive")

	saved.ActiveModelKey = ""
	saved.Dimensions = 0
	_, err = h.repo.UpdateSettings(h.ctx, saved)
	require.NoError(t, err)

	assert.Zero(t, list(repositories.ListIndexedRetrievalTenantsRequest{Limit: 10}),
		"turning retrieval off takes the organization out of the sweep")
}
