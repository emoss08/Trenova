package tenanttimezone

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
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
	orgs := mocks.NewMockOrganizationRepository(t)
	orgs.EXPECT().
		GetByID(mock.Anything, repositories.GetOrganizationByIDRequest{TenantInfo: info}).
		Return(&tenant.Organization{Timezone: "America/Chicago"}, nil)
	reader := New(Params{Organizations: orgs, Users: mocks.NewMockUserRepository(t)})

	zone, err := reader.TenantTimezone(t.Context(), info)

	require.NoError(t, err)
	assert.Equal(t, "America/Chicago", zone)
}

func TestTenantTimezone_FallsBackToThePerson(t *testing.T) {
	t.Parallel()

	info := tenantInfo()
	orgs := mocks.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tenant.Organization{Timezone: "Not/AZone"}, nil)
	users := mocks.NewMockUserRepository(t)
	users.EXPECT().
		GetByID(mock.Anything, repositories.GetUserByIDRequest{
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
	orgs := mocks.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&tenant.Organization{}, nil)

	zone, err := New(Params{Organizations: orgs, Users: mocks.NewMockUserRepository(t)}).
		TenantTimezone(t.Context(), info)

	require.NoError(t, err)
	assert.Empty(t, zone)
}

func TestTenantTimezone_SaysWhenTheOrganizationCannotBeRead(t *testing.T) {
	t.Parallel()

	orgs := mocks.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("down"))

	_, err := New(Params{Organizations: orgs, Users: mocks.NewMockUserRepository(t)}).
		TenantTimezone(t.Context(), tenantInfo())

	require.Error(t, err)
}
