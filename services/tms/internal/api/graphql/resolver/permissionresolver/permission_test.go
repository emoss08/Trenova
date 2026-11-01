package permissionresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type manifestEngine struct {
	mocks.AllowAllPermissionEngine

	userID pulid.ID
	orgID  pulid.ID
}

func (e *manifestEngine) GetLightManifest(
	_ context.Context,
	userID, orgID pulid.ID,
) (*permissionports.LightPermissionManifest, error) {
	e.userID = userID
	e.orgID = orgID
	roleID := pulid.MustNew("rol_")
	return &permissionports.LightPermissionManifest{
		Version:        "2",
		UserID:         userID,
		OrganizationID: orgID,
		ActiveRoleIDs:  []pulid.ID{roleID},
		ActiveRoles: []permissionports.RoleSummary{
			{ID: roleID, Name: "Dispatcher", PermissionCount: 3},
		},
		MaxSensitivity: permission.SensitivityInternal,
		Permissions:    map[string]uint32{"shipment": 7, "carrier": 1},
		RouteAccess:    map[string]bool{"/shipments": true, "/billing": false},
		AvailableOrgs:  []permissionports.OrgSummary{{ID: orgID, Name: "Acme"}},
		Checksum:       "abc",
		ExpiresAt:      1_800_000_000,
	}, nil
}

type allScopes struct{}

func (allScopes) Enforced(string, permission.DataScope) bool { return false }

func (allScopes) EnforceableScopes(string) []permission.DataScope {
	return []permission.DataScope{permission.DataScopeOrganization}
}

func permissionAuth() *authctx.AuthContext {
	userID := pulid.MustNew("usr_")
	return &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func TestMyPermissionsIsTheCallersOwnManifestAsLists(t *testing.T) {
	t.Parallel()

	auth := permissionAuth()
	engine := &manifestEngine{}
	resolver := &QueryResolver{
		&Deps{Core: &resolvercore.Core{L: zap.NewNop(), PermissionEngine: engine}},
	}

	manifest, err := resolver.MyPermissions(gqlctx.WithAuthContext(t.Context(), auth))

	require.NoError(t, err)
	assert.Equal(t, auth.UserID, engine.userID)
	assert.Equal(t, auth.OrganizationID, engine.orgID)
	assert.Equal(t, auth.UserID.String(), manifest.UserID)
	require.Len(t, manifest.Permissions, 2)
	assert.Equal(t, "carrier", manifest.Permissions[0].Resource)
	assert.Equal(t, 1, manifest.Permissions[0].Operations)
	assert.Equal(t, "shipment", manifest.Permissions[1].Resource)
	assert.Equal(t, 7, manifest.Permissions[1].Operations)
	require.Len(t, manifest.RouteAccess, 2)
	assert.Equal(t, "/billing", manifest.RouteAccess[0].Route)
	assert.False(t, manifest.RouteAccess[0].Allowed)
	require.Len(t, manifest.ActiveRoles, 1)
	assert.Equal(t, 3, manifest.ActiveRoles[0].PermissionCount)
	assert.Equal(t, "internal", manifest.MaxSensitivity)
	assert.Equal(t, 1_800_000_000, manifest.ExpiresAt)

	version, err := resolver.MyPermissionVersion(gqlctx.WithAuthContext(t.Context(), auth))
	require.NoError(t, err)
	assert.Equal(t, "abc", version.Checksum)
}

func TestPermissionCatalogNeedsASignedInPerson(t *testing.T) {
	t.Parallel()

	resolver := &QueryResolver{&Deps{
		Core:                  &resolvercore.Core{L: zap.NewNop()},
		PermissionRegistry:    permissioncatalog.NewRegistry(),
		PermissionScopePolicy: allScopes{},
	}}

	_, err := resolver.PermissionResources(t.Context())
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))

	agent := permissionAuth()
	agent.PrincipalType = authctx.PrincipalTypeAgent
	_, err = resolver.PermissionBundles(gqlctx.WithAuthContext(t.Context(), agent))
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestPermissionResourcesListsTheRegistryWithEnforceableScopes(t *testing.T) {
	t.Parallel()

	resolver := &QueryResolver{&Deps{
		Core:                  &resolvercore.Core{L: zap.NewNop()},
		PermissionRegistry:    permissioncatalog.NewRegistry(),
		PermissionScopePolicy: allScopes{},
	}}

	categories, err := resolver.PermissionResources(
		gqlctx.WithAuthContext(t.Context(), permissionAuth()),
	)

	require.NoError(t, err)
	require.NotEmpty(t, categories)
	found := false
	for _, category := range categories {
		for _, resource := range category.Resources {
			assert.Equal(t, []string{"organization"}, resource.EnforceableScopes)
			if resource.Resource == permission.ResourceShipment.String() {
				found = true
				assert.NotEmpty(t, resource.Operations)
			}
		}
	}
	assert.True(t, found, "the shipment resource is in the catalog")
}

func TestBundleGrantFallsBackToOrganizationForAnUnenforcedScope(t *testing.T) {
	t.Parallel()

	grant := bundleGrant(
		t.Context(),
		permissioncatalog.NewRegistry(),
		allScopes{},
		bundleGrantSource{
			resource:   permission.ResourceShipment.String(),
			operations: []permission.Operation{permission.OpRead},
			scope:      permission.DataScopeOwn,
		},
	)

	assert.Equal(t, "organization", grant.DataScope)
	assert.Equal(t, []string{"read"}, grant.Operations)
	assert.NotEqual(t, permission.ResourceShipment.String(), grant.DisplayName)
}

func TestAPIKeyAllowedResourcesNeedsAPIKeyRead(t *testing.T) {
	t.Parallel()

	engine := &resolvertest.GrantingPermissionEngine{}
	_, err := (&QueryResolver{&Deps{
		Core: &resolvercore.Core{L: zap.NewNop(), PermissionEngine: engine},
	}}).APIKeyAllowedResources(gqlctx.WithAuthContext(t.Context(), permissionAuth()))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	require.Len(t, engine.Requests, 1)
	assert.Equal(t, permission.ResourceAPIKey.String(), engine.Requests[0].Resource)
	assert.Equal(t, permission.OpRead, engine.Requests[0].Operation)
}
