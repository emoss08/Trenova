package permission

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func cachedGrant(
	scope permission.DataScope,
	ops ...permission.Operation,
) *permissionports.CachedResourcePermission {
	implied := permission.Implied(ops)
	out := make([]string, 0, len(implied))
	for _, op := range implied {
		out = append(out, string(op))
	}
	return &permissionports.CachedResourcePermission{Operations: out, DataScope: string(scope)}
}

func cachedEngine(
	t *testing.T,
	resources map[string]*permissionports.CachedResourcePermission,
) (*engine, pulid.ID, pulid.ID) {
	t.Helper()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	cacheRepo.On("Get", mock.Anything, allRolesKey(userID, orgID)).
		Return(&permissionports.CachedPermissions{
			MaxSensitivity: string(permission.SensitivityRestricted),
			ExpiresAt:      1 << 62,
			Resources:      resources,
		}, nil)

	return eng, userID, orgID
}

func TestGetResourcePermissions_AgreesWithCheckOnEveryKind(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	scenarios := []struct {
		name  string
		grant func(child, parent string) map[string]*permissionports.CachedResourcePermission
	}{
		{
			name: "the parent alone",
			grant: func(_, parent string) map[string]*permissionports.CachedResourcePermission {
				return map[string]*permissionports.CachedResourcePermission{
					parent: cachedGrant(permission.DataScopeTeam,
						registry.GetOperationsForResource(parent)...),
				}
			},
		},
		{
			name: "the kind alone",
			grant: func(child, _ string) map[string]*permissionports.CachedResourcePermission {
				return map[string]*permissionports.CachedResourcePermission{
					child: cachedGrant(permission.DataScopeTeam, permission.OpRead),
				}
			},
		},
		{
			name: "the kind's read and the parent's update, at different scopes",
			grant: func(child, parent string) map[string]*permissionports.CachedResourcePermission {
				return map[string]*permissionports.CachedResourcePermission{
					child:  cachedGrant(permission.DataScopeTeam, permission.OpRead),
					parent: cachedGrant(permission.DataScopeOrganization, permission.OpUpdate),
				}
			},
		},
		{
			name: "the kind's update and the parent's read",
			grant: func(child, parent string) map[string]*permissionports.CachedResourcePermission {
				return map[string]*permissionports.CachedResourcePermission{
					child:  cachedGrant(permission.DataScopeOrganization, permission.OpUpdate),
					parent: cachedGrant(permission.DataScopeTeam, permission.OpRead),
				}
			},
		},
	}

	checked := 0
	for _, def := range registry.All() {
		child := def.Resource
		parent := registry.GetEffectiveResource(child)
		for _, scenario := range scenarios {
			eng, userID, orgID := cachedEngine(t, scenario.grant(child, parent))
			ctx := t.Context()

			detail, err := eng.GetResourcePermissions(ctx, userID, orgID, child)
			require.NoError(t, err)
			require.NotNil(t, detail)

			for _, op := range registry.GetOperationsForResource(child) {
				result, checkErr := eng.Check(ctx, &permissionports.PermissionCheckRequest{
					UserID:         userID,
					OrganizationID: orgID,
					Resource:       child,
					Operation:      op,
				})
				require.NoError(t, checkErr)
				assert.Equal(t, result.Allowed, slices.Contains(detail.Operations, op),
					"%s:%s with %s", child, op, scenario.name)
				if result.Allowed && op == permission.OpRead {
					assert.Equal(t, result.DataScope, detail.DataScope,
						"%s read scope with %s", child, scenario.name)
				}
				checked++
			}
		}
	}
	require.Positive(t, checked)
}

func TestGetResourcePermissions_AChildReportsItsOwnGrantWithWhatItsParentImplies(t *testing.T) {
	t.Parallel()

	credential := permission.ResourceWorkerCredential.String()
	worker := permission.ResourceWorker.String()
	eng, userID, orgID := cachedEngine(t, map[string]*permissionports.CachedResourcePermission{
		credential: cachedGrant(permission.DataScopeTeam, permission.OpApprove),
		worker: cachedGrant(permission.DataScopeOrganization,
			permission.OpRead, permission.OpArchive, permission.OpRestore),
	})

	detail, err := eng.GetResourcePermissions(t.Context(), userID, orgID, credential)
	require.NoError(t, err)

	assert.Equal(t, []permission.Operation{
		permission.OpApprove,
		permission.OpArchive,
		permission.OpRead,
		permission.OpUpdate,
	}, detail.Operations,
		"the credential's own approve, what approve implies, and the worker's read and archive; "+
			"the worker's restore is no credential operation")
	assert.Equal(t, permission.DataScopeTeam, detail.DataScope,
		"a read the child's own grant answers is held to the child's scope")
}

func TestGetResourcePermissionsBatch_AgreesWithGetResourcePermissionsOnEveryKind(t *testing.T) {
	t.Parallel()

	credential := permission.ResourceWorkerCredential.String()
	worker := permission.ResourceWorker.String()
	eng, userID, orgID := cachedEngine(t, map[string]*permissionports.CachedResourcePermission{
		credential: cachedGrant(permission.DataScopeTeam, permission.OpApprove),
		worker: cachedGrant(permission.DataScopeOrganization,
			permission.OpRead, permission.OpArchive, permission.OpRestore),
	})
	ctx := t.Context()

	registry := permissioncatalog.NewRegistry()
	resources := make([]string, 0, len(registry.All()))
	for _, def := range registry.All() {
		resources = append(resources, def.Resource)
	}

	batch, err := eng.GetResourcePermissionsBatch(ctx, userID, orgID, resources)
	require.NoError(t, err)
	require.Len(t, batch, len(resources))

	for _, resource := range resources {
		single, singleErr := eng.GetResourcePermissions(ctx, userID, orgID, resource)
		require.NoError(t, singleErr)
		assert.Equal(t, single, batch[resource], resource)
	}
}
