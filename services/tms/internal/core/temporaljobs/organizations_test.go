package temporaljobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func organizations(ids ...string) []tenant.SyncOrganization {
	orgs := make([]tenant.SyncOrganization, 0, len(ids))
	for _, id := range ids {
		orgs = append(
			orgs,
			tenant.SyncOrganization{ID: pulid.ID(id), BusinessUnitID: pulid.ID("bu_" + id)},
		)
	}

	return orgs
}

func ids(page *TenantPage) []string {
	out := make([]string, 0, len(page.Tenants))
	for _, item := range page.Tenants {
		out = append(out, item.OrganizationID.String())
	}

	return out
}

// Paging walks every organization once, in id order, whatever order they were
// read in.
func TestOrganizationPage_WalksEveryOrganizationOnce(t *testing.T) {
	t.Parallel()

	orgs := organizations("org_e", "org_a", "org_d", "org_b", "org_c")

	first := OrganizationPage(orgs, nil, 2)
	assert.Equal(t, []string{"org_a", "org_b"}, ids(first))
	assert.True(t, first.HasMore)
	assert.Equal(t, pulid.ID("bu_org_a"), first.Tenants[0].BusinessUnitID)

	second := OrganizationPage(orgs, &first.Tenants[1], 2)
	assert.Equal(t, []string{"org_c", "org_d"}, ids(second))
	assert.True(t, second.HasMore)

	last := OrganizationPage(orgs, &second.Tenants[1], 2)
	assert.Equal(t, []string{"org_e"}, ids(last))
	assert.False(t, last.HasMore)
}

// An organization deleted between pages does not make the walk repeat or skip
// the ones after it.
func TestOrganizationPage_ResumesAfterACursorThatNoLongerExists(t *testing.T) {
	t.Parallel()

	page := OrganizationPage(
		organizations("org_a", "org_c", "org_d"),
		&TenantWorkItem{OrganizationID: "org_b"},
		10,
	)
	assert.Equal(t, []string{"org_c", "org_d"}, ids(page))
	assert.False(t, page.HasMore)
}

func TestOrganizationPage_EmptyAndExhausted(t *testing.T) {
	t.Parallel()

	empty := OrganizationPage(nil, nil, 10)
	require.NotNil(t, empty)
	assert.Empty(t, empty.Tenants)
	assert.False(t, empty.HasMore)

	past := OrganizationPage(organizations("org_a"), &TenantWorkItem{OrganizationID: "org_z"}, 10)
	assert.Empty(t, past.Tenants)
	assert.False(t, past.HasMore)
}
