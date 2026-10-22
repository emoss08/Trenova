package permission

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCheck_LeaveNeedsItsOwnGrant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		grants  []*permission.ResourcePermission
		checks  map[permission.Resource][]permission.Operation
		allowed map[permission.Resource][]permission.Operation
	}{
		{
			name: "worker read and update reach the worker's family but not their leave",
			grants: []*permission.ResourcePermission{{
				Resource:   permission.ResourceWorker.String(),
				Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			}},
			checks: map[permission.Resource][]permission.Operation{
				permission.ResourceWorkerLeave: {
					permission.OpRead,
					permission.OpUpdate,
					permission.OpCreate,
					permission.OpApprove,
				},
				permission.ResourceWorkerCredential: {permission.OpRead},
			},
			allowed: map[permission.Resource][]permission.Operation{
				permission.ResourceWorkerCredential: {permission.OpRead},
			},
		},
		{
			name: "worker_leave read stands on its own",
			grants: []*permission.ResourcePermission{{
				Resource:   permission.ResourceWorkerLeave.String(),
				Operations: []permission.Operation{permission.OpRead},
			}},
			checks: map[permission.Resource][]permission.Operation{
				permission.ResourceWorkerLeave: {permission.OpRead, permission.OpUpdate},
				permission.ResourceWorker:      {permission.OpRead},
			},
			allowed: map[permission.Resource][]permission.Operation{
				permission.ResourceWorkerLeave: {permission.OpRead},
			},
		},
		{
			name: "approving leave brings read and update with it",
			grants: []*permission.ResourcePermission{{
				Resource:   permission.ResourceWorkerLeave.String(),
				Operations: []permission.Operation{permission.OpApprove},
			}},
			checks: map[permission.Resource][]permission.Operation{
				permission.ResourceWorkerLeave: {
					permission.OpRead,
					permission.OpUpdate,
					permission.OpApprove,
					permission.OpManage,
				},
			},
			allowed: map[permission.Resource][]permission.Operation{
				permission.ResourceWorkerLeave: {
					permission.OpRead,
					permission.OpUpdate,
					permission.OpApprove,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
			ctx := t.Context()
			userID := pulid.MustNew("usr_")
			orgID := pulid.MustNew("org_")
			roleID := pulid.MustNew("rol_")
			for _, grant := range tt.grants {
				grant.ID = pulid.MustNew("rp_")
				grant.RoleID = roleID
				grant.DataScope = permission.DataScopeOrganization
			}

			cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil)
			cacheRepo.On("Set", ctx, allRolesKey(userID, orgID),
				mock.AnythingOfType("*permissionports.CachedPermissions"), cacheTTL).Return(nil)
			roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
				Return([]*permission.UserRoleAssignment{{
					ID:             pulid.MustNew("ura_"),
					RoleID:         roleID,
					UserID:         userID,
					OrganizationID: orgID,
				}}, nil)
			roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).
				Return([]*permission.Role{{
					ID:             roleID,
					Name:           "Role under test",
					MaxSensitivity: permission.SensitivityRestricted,
					Permissions:    tt.grants,
				}}, nil)

			for resource, ops := range tt.checks {
				for _, op := range ops {
					result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
						UserID:         userID,
						OrganizationID: orgID,
						Resource:       resource.String(),
						Operation:      op,
					})
					require.NoError(t, err)
					assert.Equal(t, slices.Contains(tt.allowed[resource], op), result.Allowed,
						"%s:%s", resource, op)
				}
			}
		})
	}
}

func TestGetResourcePermissions_LeaveReportsItsOwnGrant(t *testing.T) {
	t.Parallel()

	detail := func(t *testing.T, resources map[string]*permissionports.CachedResourcePermission) *permissionports.ResourcePermissionDetail {
		t.Helper()

		eng, _, cacheRepo, _ := setupTestEngine(t)
		ctx := t.Context()
		userID := pulid.MustNew("usr_")
		orgID := pulid.MustNew("org_")
		cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(&permissionports.CachedPermissions{
			MaxSensitivity: string(permission.SensitivityRestricted),
			ExpiresAt:      1 << 62,
			Resources:      resources,
		}, nil)

		out, err := eng.GetResourcePermissions(ctx, userID, orgID,
			permission.ResourceWorkerLeave.String())
		require.NoError(t, err)
		require.NotNil(t, out)
		return out
	}
	workerGrant := &permissionports.CachedResourcePermission{
		Operations: []string{string(permission.OpRead), string(permission.OpUpdate)},
		DataScope:  string(permission.DataScopeOrganization),
	}

	assert.Empty(t, detail(t, map[string]*permissionports.CachedResourcePermission{
		permission.ResourceWorker.String(): workerGrant,
	}).Operations, "a worker grant says nothing about what the holder may do with leave")

	assert.Equal(t, []permission.Operation{permission.OpRead},
		detail(t, map[string]*permissionports.CachedResourcePermission{
			permission.ResourceWorker.String(): workerGrant,
			permission.ResourceWorkerLeave.String(): {
				Operations: []string{string(permission.OpRead)},
				DataScope:  string(permission.DataScopeOrganization),
			},
		}).Operations)
}
