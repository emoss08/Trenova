package tenanttimezone

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func TestTenantTimezone_ReadsTheOrganizationFirst(t *testing.T) {
	t.Parallel()

	info := tenantInfo()
	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().
		GetByID(mock.Anything, tenantports.GetOrganizationByIDRequest{TenantInfo: info}).
		Return(&tenant.Organization{Timezone: "America/Chicago"}, nil)
	reader := New(Params{Organizations: orgs, Users: tenantportsmock.NewMockUserRepository(t)})

	zone, err := reader.TenantTimezone(t.Context(), info)

	require.NoError(t, err)
	assert.Equal(t, "America/Chicago", zone)
}

func TestTenantTimezone_FallsBackToThePerson(t *testing.T) {
	t.Parallel()

	info := tenantInfo()
	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tenant.Organization{Timezone: "Not/AZone"}, nil)
	users := tenantportsmock.NewMockUserRepository(t)
	users.EXPECT().
		GetByID(mock.Anything, tenantports.GetUserByIDRequest{
			TenantInfo:   info,
			LookupUserID: info.UserID,
		}).
		Return(&tenant.User{Timezone: "America/Denver"}, nil)

	zone, err := New(Params{Organizations: orgs, Users: users}).TenantTimezone(t.Context(), info)

	require.NoError(t, err)
	assert.Equal(t, "America/Denver", zone)
}

func TestTenantTimezone_NamesNoZoneWhenNeitherHasOne(t *testing.T) {
	t.Parallel()

	info := tenantInfo()
	info.UserID = pulid.Nil
	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&tenant.Organization{}, nil)

	zone, err := New(Params{Organizations: orgs, Users: tenantportsmock.NewMockUserRepository(t)}).
		TenantTimezone(t.Context(), info)

	require.NoError(t, err)
	assert.Empty(t, zone)
}

func TestTenantTimezone_SaysWhenTheOrganizationCannotBeRead(t *testing.T) {
	t.Parallel()

	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("down"))

	_, err := New(Params{Organizations: orgs, Users: tenantportsmock.NewMockUserRepository(t)}).
		TenantTimezone(t.Context(), tenantInfo())

	require.Error(t, err)
}
