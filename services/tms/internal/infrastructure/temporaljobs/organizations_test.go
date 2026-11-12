package temporaljobs

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sortedTenants pages like the repository: id order, after the cursor.
type sortedTenants struct {
	tenants []pagination.TenantInfo
	limits  []int
}

func newSortedTenants(ids ...string) *sortedTenants {
	tenants := make([]pagination.TenantInfo, 0, len(ids))
	for _, id := range ids {
		tenants = append(tenants, pagination.TenantInfo{OrgID: pulid.ID(id), BuID: pulid.ID("bu_" + id)})
	}
	slices.SortFunc(tenants, func(a, b pagination.TenantInfo) int {
		return strings.Compare(a.OrgID.String(), b.OrgID.String())
	})

	return &sortedTenants{tenants: tenants}
}

func (s *sortedTenants) ListTenantsPage(
	_ context.Context,
	after pulid.ID,
	limit int,
) ([]pagination.TenantInfo, error) {
	s.limits = append(s.limits, limit)
	out := make([]pagination.TenantInfo, 0, limit)
	for _, tenant := range s.tenants {
		if after.IsNotNil() && tenant.OrgID.String() <= after.String() {
			continue
		}
		if len(out) == limit {
			break
		}
		out = append(out, tenant)
	}

	return out, nil
}

func ids(page *TenantPage) []string {
	out := make([]string, 0, len(page.Tenants))
	for _, item := range page.Tenants {
		out = append(out, item.OrganizationID.String())
	}

	return out
}

// Paging walks every organization once, in id order, reading one past the
// page to know whether another follows.
func TestOrganizationPage_WalksEveryOrganizationOnce(t *testing.T) {
	t.Parallel()

	tenants := newSortedTenants("org_e", "org_a", "org_d", "org_b", "org_c")

	first, err := OrganizationPage(t.Context(), tenants, nil, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"org_a", "org_b"}, ids(first))
	assert.True(t, first.HasMore)

	last := first.Tenants[len(first.Tenants)-1]
	second, err := OrganizationPage(t.Context(), tenants, &last, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"org_c", "org_d"}, ids(second))
	assert.True(t, second.HasMore)

	last = second.Tenants[len(second.Tenants)-1]
	third, err := OrganizationPage(t.Context(), tenants, &last, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"org_e"}, ids(third))
	assert.False(t, third.HasMore)

	assert.Equal(t, []int{3, 3, 3}, tenants.limits)
	assert.Equal(t, pulid.ID("bu_org_a"), first.Tenants[0].BusinessUnitID)
}

// A page that ends exactly at the last organization says nothing follows.
func TestOrganizationPage_ExactPageHasNoMore(t *testing.T) {
	t.Parallel()

	page, err := OrganizationPage(t.Context(), newSortedTenants("org_a", "org_b"), nil, 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"org_a", "org_b"}, ids(page))
	assert.False(t, page.HasMore)
}
