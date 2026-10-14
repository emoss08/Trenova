//go:build integration

package userrepository

import (
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newUserForCreate(data *seedtest.TestData, username, email string) *tenant.User {
	return &tenant.User{
		BusinessUnitID:        data.BusinessUnit.ID,
		CurrentOrganizationID: data.Organization.ID,
		Status:                domaintypes.StatusActive,
		Name:                  "Dana Whitfield",
		Username:              username,
		EmailAddress:          email,
		Timezone:              "America/Chicago",
		TimeFormat:            domaintypes.TimeFormat12Hour,
		Locale:                "en",
		AvatarStyle:           tenant.AvatarStyleAuto,
		AvailabilityStatus:    tenant.AvailabilityAvailable,
		Password:              "$2a$10$abcdefghijklmnopqrstuuJ0e8qY3s6a4uI2k0p0c4m1b8xG5r1aG",
	}
}

func TestCreateInsertsTheUserAndADefaultMembership(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	data := seedtest.SeedFullTestData(t, ctx, db)

	created, err := repo.Create(ctx, tenantports.CreateUserRequest{
		User: newUserForCreate(data, "dwhitfield", "dana.whitfield@example.com"),
		Membership: &tenant.OrganizationMembership{
			BusinessUnitID: data.BusinessUnit.ID,
			OrganizationID: data.Organization.ID,
			GrantedByID:    data.User.ID,
			IsDefault:      true,
		},
	})
	require.NoError(t, err)
	require.True(t, created.ID.IsNotNil())

	loaded, err := repo.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  data.Organization.ID,
			BuID:   data.BusinessUnit.ID,
			UserID: data.User.ID,
		},
		LookupUserID: created.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, "dwhitfield", loaded.Username)
	assert.Nil(t, loaded.LastLoginAt)
	assert.Nil(t, loaded.PasswordChangedAt)
	assert.True(t, loaded.AwaitingAccountSetup())

	memberships, err := repo.ListOrganizationMemberships(ctx, created.ID, data.BusinessUnit.ID)
	require.NoError(t, err)
	require.Len(t, memberships, 1)
	assert.Equal(t, data.Organization.ID, memberships[0].OrganizationID)
	assert.Equal(t, data.User.ID, memberships[0].GrantedByID)
	assert.True(t, memberships[0].IsDefault)
}

func TestCreateReportsADuplicateEmailAsAFieldError(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	data := seedtest.SeedFullTestData(t, ctx, db)

	_, err := repo.Create(ctx, tenantports.CreateUserRequest{
		User: newUserForCreate(data, "someoneelse", strings.ToUpper(data.User.EmailAddress)),
		Membership: &tenant.OrganizationMembership{
			BusinessUnitID: data.BusinessUnit.ID,
			OrganizationID: data.Organization.ID,
			GrantedByID:    data.User.ID,
			IsDefault:      true,
		},
	})

	multiErr, ok := errors.AsType[*errortypes.MultiError](err)
	require.True(t, ok, "expected a field error, got %v", err)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "emailAddress", multiErr.Errors[0].Field)
	assert.Equal(t, errortypes.ErrDuplicate, multiErr.Errors[0].Code)
}

func TestFindIdentityConflictsMatchesTheUniqueIndexes(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	data := seedtest.SeedFullTestData(t, ctx, db)

	conflicts, err := repo.FindIdentityConflicts(ctx, tenantports.UserIdentityConflictsRequest{
		EmailAddress: strings.ToUpper(data.User.EmailAddress),
		Username:     strings.ToUpper(data.User.Username),
	})
	require.NoError(t, err)
	assert.True(t, conflicts.EmailAddress)
	assert.True(t, conflicts.Username)

	conflicts, err = repo.FindIdentityConflicts(ctx, tenantports.UserIdentityConflictsRequest{
		EmailAddress:  data.User.EmailAddress,
		Username:      data.User.Username,
		ExcludeUserID: data.User.ID,
	})
	require.NoError(t, err)
	assert.False(t, conflicts.EmailAddress, "a user does not conflict with themselves")
	assert.False(t, conflicts.Username)

	conflicts, err = repo.FindIdentityConflicts(ctx, tenantports.UserIdentityConflictsRequest{
		EmailAddress: "nobody@example.com",
		Username:     "nobody",
	})
	require.NoError(t, err)
	assert.False(t, conflicts.EmailAddress)
	assert.False(t, conflicts.Username)
}
