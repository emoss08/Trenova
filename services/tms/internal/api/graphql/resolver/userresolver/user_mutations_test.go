package userresolver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/app/passwordresetservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/identitytest"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingRoles struct {
	requests []roleservice.AssignRoleRequest
}

func (r *recordingRoles) AssignRole(_ context.Context, req roleservice.AssignRoleRequest) error {
	r.requests = append(r.requests, req)
	return nil
}

func (r *recordingRoles) RevokeOrganizationRoles(
	context.Context,
	roleservice.RevokeOrganizationRolesRequest,
) error {
	return nil
}

type recordingInvites struct {
	requests []passwordresetservice.InviteRequest
}

func (r *recordingInvites) SendInvite(
	_ context.Context,
	req passwordresetservice.InviteRequest,
) error {
	r.requests = append(r.requests, req)
	return nil
}

type harness struct {
	repo        *tenantportsmock.MockUserRepository
	audit       *auditportsmock.MockAuditService
	roles       *recordingRoles
	invites     *recordingInvites
	permissions *resolvertest.GrantingPermissionEngine
	auth        *authctx.AuthContext
	deps        *Deps
}

func newHarness(t *testing.T, granted ...string) *harness {
	t.Helper()

	repo := tenantportsmock.NewMockUserRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
	roles := &recordingRoles{}
	invites := &recordingInvites{}
	grants := make(map[string]bool, len(granted))
	for _, grant := range granted {
		grants[grant] = true
	}
	permissions := &resolvertest.GrantingPermissionEngine{Granted: grants}

	service := identitytest.NewUserServiceForTest(identitytest.UserServiceDeps{
		Users:   repo,
		Audit:   audit,
		Roles:   roles,
		Invites: invites,
	})

	userID := pulid.MustNew("usr_")
	return &harness{
		repo:        repo,
		audit:       audit,
		roles:       roles,
		invites:     invites,
		permissions: permissions,
		auth: &authctx.AuthContext{
			PrincipalType:  authctx.PrincipalTypeUser,
			PrincipalID:    userID,
			UserID:         userID,
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		deps: &Deps{
			Core:        &resolvercore.Core{L: zap.NewNop(), PermissionEngine: permissions},
			UserService: service,
		},
	}
}

func (h *harness) ctx(t *testing.T) context.Context {
	t.Helper()
	return gqlctx.WithAuthContext(t.Context(), h.auth)
}

func grant(op permission.Operation) string {
	return permission.ResourceUser.String() + "|" + string(op)
}

func TestCreateUserRequiresCreatePermissionOnUsers(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	resolver := &MutationResolver{h.deps}

	_, err := resolver.CreateUser(h.ctx(t), gqlmodel.CreateUserInput{
		Name:         "Dana Whitfield",
		Username:     "dwhitfield",
		EmailAddress: "dana@example.com",
		Timezone:     "America/Chicago",
		Status:       domaintypes.StatusActive,
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	require.Len(t, h.permissions.Requests, 1)
	assert.Equal(t, permission.ResourceUser.String(), h.permissions.Requests[0].Resource)
	assert.Equal(t, permission.OpCreate, h.permissions.Requests[0].Operation)
	h.repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCreateUserMapsTheInputAndInvites(t *testing.T) {
	t.Parallel()

	h := newHarness(t, grant(permission.OpCreate))
	resolver := &MutationResolver{h.deps}
	roleID := pulid.MustNew("rol_")
	twentyFour := gqlmodel.TimeFormatTwentyFourHour
	locale := "es"
	mustChange := true

	h.repo.EXPECT().FindIdentityConflicts(mock.Anything, mock.Anything).
		Return(tenantports.UserIdentityConflicts{}, nil).Once()
	h.repo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(req tenantports.CreateUserRequest) bool {
			user := req.User
			return user.Name == "Dana Whitfield" &&
				user.Username == "dwhitfield" &&
				user.EmailAddress == "dana@example.com" &&
				user.Timezone == "America/Chicago" &&
				user.TimeFormat == domaintypes.TimeFormat24Hour &&
				user.Locale == "es" &&
				user.MustChangePassword &&
				user.CurrentOrganizationID == h.auth.OrganizationID &&
				user.BusinessUnitID == h.auth.BusinessUnitID &&
				user.Password != ""
		})).
		RunAndReturn(func(_ context.Context, req tenantports.CreateUserRequest) (*tenant.User, error) {
			req.User.ID = pulid.MustNew("usr_")
			return req.User, nil
		}).
		Once()
	h.audit.On("LogAction", mock.Anything, mock.Anything).Return(nil).Once()

	created, err := resolver.CreateUser(h.ctx(t), gqlmodel.CreateUserInput{
		Name:               "Dana Whitfield",
		Username:           "dwhitfield",
		EmailAddress:       "Dana@Example.com",
		Timezone:           "America/Chicago",
		TimeFormat:         &twentyFour,
		Locale:             &locale,
		Status:             domaintypes.StatusActive,
		MustChangePassword: &mustChange,
		RoleIds:            []string{roleID.String()},
	})

	require.NoError(t, err)
	require.NotNil(t, created)
	require.Len(t, h.roles.requests, 1)
	assert.Equal(t, roleID, h.roles.requests[0].Assignment.RoleID)
	assert.Equal(t, created.ID, h.roles.requests[0].Assignment.UserID)
	require.Len(t, h.invites.requests, 1)
	assert.Equal(t, created.ID, h.invites.requests[0].User.ID)
}

func TestCreateUserRefusesAMalformedRoleID(t *testing.T) {
	t.Parallel()

	h := newHarness(t, grant(permission.OpCreate))
	resolver := &MutationResolver{h.deps}

	_, err := resolver.CreateUser(h.ctx(t), gqlmodel.CreateUserInput{
		Name:         "Dana Whitfield",
		Username:     "dwhitfield",
		EmailAddress: "dana@example.com",
		Timezone:     "America/Chicago",
		Status:       domaintypes.StatusActive,
		RoleIds:      []string{"not-an-id"},
	})

	require.Error(t, err)
	h.repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func targetUser(h *harness) *tenant.User {
	return &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        h.auth.BusinessUnitID,
		CurrentOrganizationID: h.auth.OrganizationID,
		Status:                domaintypes.StatusActive,
		Name:                  "Target User",
		Username:              "target",
		EmailAddress:          "target@example.com",
		Timezone:              "America/Chicago",
		TimeFormat:            domaintypes.TimeFormat12Hour,
		IsLocked:              true,
		Version:               4,
	}
}

func TestPatchUserRequiresUpdatePermissionOnUsers(t *testing.T) {
	t.Parallel()

	h := newHarness(t, grant(permission.OpCreate))
	resolver := &MutationResolver{h.deps}

	_, err := resolver.PatchUser(
		h.ctx(t),
		pulid.MustNew("usr_").String(),
		gqlmodel.UserPatchInput{},
	)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Equal(t, permission.OpUpdate, h.permissions.Requests[0].Operation)
	h.repo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestPatchUserWritesOnlyWhatWasSent(t *testing.T) {
	t.Parallel()

	h := newHarness(t, grant(permission.OpUpdate))
	resolver := &MutationResolver{h.deps}
	target := targetUser(h)
	unlocked := false
	twentyFour := gqlmodel.TimeFormatTwentyFourHour

	h.repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req tenantports.GetUserByIDRequest) bool {
			return req.LookupUserID == target.ID && req.TenantInfo.UserID == h.auth.UserID
		})).
		Return(target, nil).
		Twice()
	h.repo.EXPECT().FindIdentityConflicts(mock.Anything, mock.Anything).
		Return(tenantports.UserIdentityConflicts{}, nil).Once()
	h.repo.EXPECT().
		UpdateColumns(mock.Anything, mock.MatchedBy(func(req tenantports.UpdateUserColumnsRequest) bool {
			return assert.ObjectsAreEqual([]string{"time_format", "is_locked"}, req.Columns) &&
				!req.Entity.IsLocked &&
				req.Entity.TimeFormat == domaintypes.TimeFormat24Hour &&
				req.Entity.Name == "Target User"
		})).
		RunAndReturn(func(_ context.Context, req tenantports.UpdateUserColumnsRequest) (*tenant.User, error) {
			return req.Entity, nil
		}).
		Once()
	h.audit.On("LogAction", mock.Anything, mock.Anything).Return(nil).Once()

	updated, err := resolver.PatchUser(h.ctx(t), target.ID.String(), gqlmodel.UserPatchInput{
		IsLocked:   graphql.OmittableOf(&unlocked),
		TimeFormat: graphql.OmittableOf(&twentyFour),
	})

	require.NoError(t, err)
	require.NotNil(t, updated)
}

func TestPatchUserRefusesANullTimeFormat(t *testing.T) {
	t.Parallel()

	h := newHarness(t, grant(permission.OpUpdate))
	resolver := &MutationResolver{h.deps}
	target := targetUser(h)

	h.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(target, nil).Once()

	_, err := resolver.PatchUser(h.ctx(t), target.ID.String(), gqlmodel.UserPatchInput{
		TimeFormat: graphql.OmittableOf[*gqlmodel.TimeFormat](nil),
	})

	resolvertest.RequireRequiredPatchError(t, err, "timeFormat")
	h.repo.AssertNotCalled(t, "UpdateColumns", mock.Anything, mock.Anything)
}
