package aianalyticsjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fixedActive struct {
	tenants []pagination.TenantInfo
	since   int64
}

func (f *fixedActive) ListActiveTenants(_ context.Context, since int64, _ int) ([]pagination.TenantInfo, error) {
	f.since = since

	return f.tenants, nil
}

func tenantsOf(ids ...string) []pagination.TenantInfo {
	out := make([]pagination.TenantInfo, 0, len(ids))
	for _, id := range ids {
		out = append(out, pagination.TenantInfo{OrgID: pulid.ID(id), BuID: pulid.ID("bu_" + id)})
	}

	return out
}

func orgIDs(page *temporaljobs.TenantPage) []string {
	out := make([]string, 0, len(page.Tenants))
	for _, item := range page.Tenants {
		out = append(out, item.OrganizationID.String())
	}

	return out
}

// The refresh visits only the organizations with AI activity in the last
// hour, paged in id order.
func TestListAIAnalyticsOrganizationsPagesTheActiveTenants(t *testing.T) {
	t.Parallel()

	active := &fixedActive{tenants: tenantsOf("org_a", "org_b", "org_c")}
	a := NewActivities(ActivitiesParams{Active: active, Logger: zap.NewNop()})
	a.now = func() int64 { return 10_000 }

	first, err := a.ListAIAnalyticsOrganizationsActivity(t.Context(), &ListAIAnalyticsOrganizationsInput{Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, []string{"org_a", "org_b"}, orgIDs(first))
	assert.True(t, first.HasMore)
	assert.Equal(t, int64(10_000)-int64(activeWindow.Seconds()), active.since)

	last := first.Tenants[len(first.Tenants)-1]
	second, err := a.ListAIAnalyticsOrganizationsActivity(t.Context(), &ListAIAnalyticsOrganizationsInput{
		After: &last,
		Limit: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"org_c"}, orgIDs(second))
	assert.False(t, second.HasMore)
}

// A cursor whose organization fell out of the active set still pages from
// the right place.
func TestPageAfterACursorNoLongerInTheSet(t *testing.T) {
	t.Parallel()

	cursor := temporaljobs.NewTenantWorkItem(pagination.TenantInfo{OrgID: "org_b", BuID: "bu_org_b"}, 1)
	page := pageAfter(tenantsOf("org_a", "org_c", "org_d"), &cursor, 10)

	assert.Equal(t, []string{"org_c", "org_d"}, orgIDs(page))
	assert.False(t, page.HasMore)
}
