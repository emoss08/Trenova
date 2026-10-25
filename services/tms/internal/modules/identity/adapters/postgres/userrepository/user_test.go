package userrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantUsersExcludesTheSystemUserAndOtherOrganizations(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, tenantUsers(keeltest.TestOrganizationID).Limit(10))

	assert.Contains(t, sql, `"current_organization_id" = `)
	assert.Contains(t, sql, `"username" <> `)
	assert.Contains(t, sql, `"business_unit_id" = `)
	assert.Contains(t, args, string(keeltest.TestOrganizationID))
	assert.Contains(t, args, tenant.SystemUsername)
}

func TestMembershipReadScopeMirrorsRowLevelSecurity(t *testing.T) {
	t.Parallel()

	userID := pulid.ID("usr_1")
	ctx := dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: keeltest.TestOrganizationID,
		BusinessUnitID: keeltest.TestBusinessUnitID,
		UserID:         userID,
	})

	readCtx, visible := membershipReadScope(ctx)
	require.True(t, dbscope.IsSystem(readCtx))
	require.NotNil(t, visible)

	m := tenantdb.OrganizationMemberships
	sql, args := keeltest.RenderIn(t, readCtx, allTenantsWhenSystem(readCtx, m.Where(visible)).Limit(10))

	assert.Contains(t, sql, `"organization_id" = `)
	assert.Contains(t, sql, `"user_id" = `)
	assert.Contains(t, sql, " OR ")
	assert.Contains(t, args, string(keeltest.TestOrganizationID))
	assert.Contains(t, args, string(keeltest.TestBusinessUnitID))
	assert.Contains(t, args, string(userID))
}

func TestMembershipReadScopeLeavesASystemScopeUnchanged(t *testing.T) {
	t.Parallel()

	ctx := dbscope.WithSystem(t.Context(), "test")
	readCtx, visible := membershipReadScope(ctx)

	assert.Equal(t, ctx, readCtx)
	assert.Nil(t, visible)
}

func TestUpdatableUserColumnsCoverEveryColumnTheServicesWrite(t *testing.T) {
	t.Parallel()

	for _, column := range []string{
		"status", "name", "email_address", "timezone", "time_format", "locale", "is_locked",
		"must_change_password", "job_title", "phone_number", "avatar_style", "profile_pic_url",
		"thumbnail_url", "assistant_hidden", "availability_status", "focus_until", "away_back_on",
		"away_delegate_user_id", "away_note",
	} {
		assert.Contains(t, updatableUserColumns, column)
	}
	assert.NotContains(t, updatableUserColumns, "password")
	assert.NotContains(t, updatableUserColumns, "business_unit_id")
}
