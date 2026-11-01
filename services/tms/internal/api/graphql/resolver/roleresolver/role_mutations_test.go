package roleresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/internal/testutil/permchangetest"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func roleAuth() *authctx.AuthContext {
	userID := pulid.MustNew("usr_")
	return &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func roleDeps(
	t *testing.T,
	engine permissionports.PermissionEngine,
	repo permissionports.RoleRepository,
) *Deps {
	t.Helper()

	return &Deps{
		Core: &resolvercore.Core{L: zap.NewNop(), PermissionEngine: engine},
		RoleService: roleservice.New(roleservice.Params{
			Logger:           zap.NewNop(),
			RoleRepo:         repo,
			Notifier:         &permchangetest.Recorder{},
			PermissionEngine: engine,
			Validator:        roleservice.NewTestValidator(),
			Registry:         permissioncatalog.NewEmptyRegistry(),
		}),
	}
}

func TestRoleResolversCheckTheirOwnPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("rol_").String()
	input := gqlmodel.RoleInput{Name: "Dispatcher", MaxSensitivity: "internal"}
	cases := []struct {
		name     string
		resource permission.Resource
		op       permission.Operation
		call     func(ctx context.Context, d *Deps) error
	}{
		{
			"createRole",
			permission.ResourceRole,
			permission.OpCreate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).CreateRole(ctx, input)
				return err
			},
		},
		{
			"updateRole",
			permission.ResourceRole,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).UpdateRole(ctx, id, input)
				return err
			},
		},
		{
			"deleteRole",
			permission.ResourceRole,
			permission.OpDelete,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).DeleteRole(ctx, id, nil)
				return err
			},
		},
		{
			"assignRole",
			permission.ResourceUser,
			permission.OpAssign,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).AssignRole(ctx, id, gqlmodel.AssignRoleInput{
					UserID: pulid.MustNew("usr_").String(),
				})
				return err
			},
		},
		{
			"unassignRole",
			permission.ResourceUser,
			permission.OpUnassign,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).UnassignRole(ctx, pulid.MustNew("ura_").String())
				return err
			},
		},
		{
			"rolePermissions",
			permission.ResourceRole,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := (&QueryResolver{d}).RolePermissions(ctx, id)
				return err
			},
		},
		{
			"roleImpact",
			permission.ResourceRole,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := (&QueryResolver{d}).RoleImpact(ctx, id)
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := &resolvertest.GrantingPermissionEngine{}
			repo := permissionportsmock.NewMockRoleRepository(t)
			ctx := gqlctx.WithAuthContext(t.Context(), roleAuth())

			err := tc.call(ctx, roleDeps(t, engine, repo))

			require.Error(t, err)
			assert.True(t, errortypes.IsAuthorizationError(err))
			require.Len(t, engine.Requests, 1)
			assert.Equal(t, tc.resource.String(), engine.Requests[0].Resource)
			assert.Equal(t, tc.op, engine.Requests[0].Operation)
		})
	}
}

func TestRoleFromInputMapsTheGrantsAndKeepsAbsentOnesUntouched(t *testing.T) {
	t.Parallel()

	parent := pulid.MustNew("rol_")
	description := "Plans loads"
	responsibility := "Operations"

	role, err := roleFromInput(&gqlmodel.RoleInput{
		Name:               "Dispatcher",
		Description:        &description,
		CoreResponsibility: &responsibility,
		MaxSensitivity:     "restricted",
		ParentRoleIds:      []string{parent.String()},
		Permissions: []*gqlmodel.ResourcePermissionInput{
			{Resource: "shipment", Operations: []string{"read", "update"}, DataScope: "own"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Dispatcher", role.Name)
	assert.Equal(t, "Plans loads", role.Description)
	assert.Equal(t, rbac.CoreResponsibilityOperations, role.CoreResponsibility)
	assert.Equal(t, permission.SensitivityRestricted, role.MaxSensitivity)
	assert.Equal(t, []pulid.ID{parent}, role.ParentRoleIDs)
	require.Len(t, role.Permissions, 1)
	assert.Equal(t, "shipment", role.Permissions[0].Resource)
	assert.Equal(
		t,
		[]permission.Operation{permission.OpRead, permission.OpUpdate},
		role.Permissions[0].Operations,
	)
	assert.Equal(t, permission.DataScopeOwn, role.Permissions[0].DataScope)

	untouched, err := roleFromInput(&gqlmodel.RoleInput{Name: "Clerk", MaxSensitivity: "internal"})
	require.NoError(t, err)
	assert.Nil(t, untouched.Permissions)
	assert.Nil(t, untouched.ParentRoleIDs)

	_, err = roleFromInput(&gqlmodel.RoleInput{
		Name:           "Clerk",
		MaxSensitivity: "internal",
		ParentRoleIds:  []string{"bad"},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestRolePermissionsReturnsTheGrantsOfARoleInTheCallersOrganization(t *testing.T) {
	t.Parallel()

	auth := roleAuth()
	roleID := pulid.MustNew("rol_")
	engine := &resolvertest.GrantingPermissionEngine{
		Granted: map[string]bool{
			permission.ResourceRole.String() + "|" + string(permission.OpRead): true,
		},
	}
	repo := permissionportsmock.NewMockRoleRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req permissionports.GetRoleByIDRequest) bool {
			return req.ID == roleID && req.TenantInfo.OrgID == auth.OrganizationID
		})).
		Return(&rbac.Role{
			ID: roleID,
			Permissions: []*rbac.ResourcePermission{
				{Resource: "shipment", Operations: []permission.Operation{permission.OpRead}},
			},
		}, nil).
		Once()

	resolver := &QueryResolver{roleDeps(t, engine, repo)}
	grants, err := resolver.RolePermissions(
		gqlctx.WithAuthContext(t.Context(), auth),
		roleID.String(),
	)

	require.NoError(t, err)
	require.Len(t, grants, 1)
	assert.Equal(t, "shipment", grants[0].Resource)

	ops, err := (&ResourcePermissionResolver{resolver.Deps}).Operations(t.Context(), grants[0])
	require.NoError(t, err)
	assert.Equal(t, []string{"read"}, ops)
}

func TestRoleImpactRefusesARoleOutsideTheCallersOrganization(t *testing.T) {
	t.Parallel()

	auth := roleAuth()
	engine := &resolvertest.GrantingPermissionEngine{
		Granted: map[string]bool{
			permission.ResourceRole.String() + "|" + string(permission.OpRead): true,
		},
	}
	repo := permissionportsmock.NewMockRoleRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Role not found")).
		Once()

	_, err := (&QueryResolver{roleDeps(t, engine, repo)}).RoleImpact(
		gqlctx.WithAuthContext(t.Context(), auth),
		pulid.MustNew("rol_").String(),
	)

	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	repo.AssertNotCalled(t, "GetUsersWithRole", mock.Anything, mock.Anything)
}
