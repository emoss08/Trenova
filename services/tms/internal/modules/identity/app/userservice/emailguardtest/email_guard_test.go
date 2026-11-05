package emailguardtest

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/identitytest"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type held map[pulid.ID]map[string]permissionports.EffectiveResourcePermission

type heldPermissions struct {
	permissionports.PermissionEngine

	byUser held
}

func (h *heldPermissions) GetEffectivePermissions(
	_ context.Context,
	userID, _ pulid.ID,
) (*permissionports.EffectivePermissions, error) {
	return &permissionports.EffectivePermissions{Resources: h.byUser[userID]}, nil
}

func administration() map[string]permissionports.EffectiveResourcePermission {
	return map[string]permissionports.EffectiveResourcePermission{
		permission.ResourceRole.String(): {
			Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			DataScope:  permission.DataScopeAll,
		},
	}
}

type fixture struct {
	service *userservice.Service
	repo    *tenantportsmock.MockUserRepository
	target  *tenant.User
	actor   pagination.TenantInfo
}

func newFixture(t *testing.T, byUser func(actor, target pulid.ID) held) *fixture {
	t.Helper()

	repo := tenantportsmock.NewMockUserRepository(t)
	target := &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
		CurrentOrganizationID: pulid.MustNew("org_"),
		Status:                domaintypes.StatusActive,
		Name:                  "Target User",
		Username:              "target",
		EmailAddress:          "target@example.com",
		Timezone:              "America/New_York",
		TimeFormat:            domaintypes.TimeFormat24Hour,
	}
	actor := pagination.TenantInfo{
		OrgID:  target.CurrentOrganizationID,
		BuID:   target.BusinessUnitID,
		UserID: pulid.MustNew("usr_"),
	}
	repo.On("GetByID", mock.Anything, mock.Anything).Return(target, nil)
	repo.On("ListOrganizationMemberships", mock.Anything, mock.Anything, mock.Anything).
		Return([]*tenant.OrganizationMembership{}, nil).Maybe()

	return &fixture{
		service: identitytest.NewUserServiceForTest(identitytest.UserServiceDeps{
			Users: repo,
			Audit: &mocks.NoopAuditService{},
			Held:  &heldPermissions{byUser: byUser(actor.UserID, target.ID)},
		}),
		repo:   repo,
		target: target,
		actor:  actor,
	}
}

func (f *fixture) expectSave() *tenantports.UpdateUserColumnsRequest {
	captured := &tenantports.UpdateUserColumnsRequest{}
	f.repo.On("FindIdentityConflicts", mock.Anything, mock.Anything).
		Return(tenantports.UserIdentityConflicts{}, nil).Maybe()
	f.repo.On("UpdateColumns", mock.Anything, mock.Anything).
		Return(func(_ context.Context, req tenantports.UpdateUserColumnsRequest) *tenant.User {
			*captured = req
			saved := *req.Entity

			return &saved
		}, func(context.Context, tenantports.UpdateUserColumnsRequest) error {
			return nil
		}).Once()

	return captured
}

func (f *fixture) patch(
	t *testing.T,
	apply func(*tenant.User) error,
) (*tenant.User, error) {
	t.Helper()

	return f.service.Patch(t.Context(), &userservice.PatchUserRequest{
		TenantInfo: f.actor,
		UserID:     f.target.ID,
		Apply:      apply,
	})
}

func setEmail(address string) func(*tenant.User) error {
	return func(user *tenant.User) error {
		user.EmailAddress = address

		return nil
	}
}

func TestPatch_RefusesAnEmailChangeOnAnAccountBeyondTheActorsOwnAccess(t *testing.T) {
	t.Parallel()

	f := newFixture(t, func(_, target pulid.ID) held {
		return held{target: administration()}
	})

	_, err := f.patch(t, setEmail("attacker@example.com"))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err), err.Error())
	f.repo.AssertNotCalled(t, "UpdateColumns", mock.Anything, mock.Anything)
}

func TestPatch_LetsAnActorWhoCoversTheTargetChangeTheirEmail(t *testing.T) {
	t.Parallel()

	f := newFixture(t, func(actor, target pulid.ID) held {
		return held{actor: administration(), target: administration()}
	})
	captured := f.expectSave()

	_, err := f.patch(t, setEmail("new@example.com"))

	require.NoError(t, err)
	assert.Equal(t, "new@example.com", captured.Entity.EmailAddress)
}

func TestPatch_LetsAPersonChangeTheirOwnEmail(t *testing.T) {
	t.Parallel()

	f := newFixture(t, func(_, target pulid.ID) held {
		return held{target: administration()}
	})
	f.actor.UserID = f.target.ID
	captured := f.expectSave()

	_, err := f.patch(t, setEmail("mine@example.com"))

	require.NoError(t, err)
	assert.Equal(t, "mine@example.com", captured.Entity.EmailAddress)
}

func TestPatch_LeavesOtherFieldsAloneWhenTheActorDoesNotCoverTheTarget(t *testing.T) {
	t.Parallel()

	f := newFixture(t, func(_, target pulid.ID) held {
		return held{target: administration()}
	})
	captured := f.expectSave()

	_, err := f.patch(t, func(user *tenant.User) error {
		user.JobTitle = "Dispatcher"

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, "Dispatcher", captured.Entity.JobTitle)
}
