package permission

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/supportaccessservice"
	permissiondomain "github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type grantedSupport struct {
	registry *permissioncatalog.Registry
	scope    supportaccess.Scope
	live     bool
}

func (g *grantedSupport) Permissions(
	context.Context,
	pulid.ID,
	pulid.ID,
) (*permissionports.CachedPermissions, error) {
	if !g.live {
		return &permissionports.CachedPermissions{Resources: map[string]*permissionports.CachedResourcePermission{}}, nil
	}
	return &permissionports.CachedPermissions{
		MaxSensitivity: string(permissiondomain.SensitivityConfidential),
		Resources:      supportaccessservice.SupportPermissions(g.registry, g.scope),
	}, nil
}

func (g *grantedSupport) RequireLive(context.Context, *supportaccessports.RequireLiveRequest) error {
	return nil
}

func everyRegistryCheck(registry *permissioncatalog.Registry) []permissionports.ResourceOperationCheck {
	var checks []permissionports.ResourceOperationCheck
	for _, def := range registry.All() {
		for _, op := range def.Operations {
			checks = append(checks, permissionports.ResourceOperationCheck{
				Resource:  def.Resource,
				Operation: op.Operation,
			})
		}
	}
	return checks
}

func checkAsSupport(
	t *testing.T,
	eng *engine,
	principal, orgID pulid.ID,
) (*permissionports.BatchPermissionCheckResult, []permissionports.ResourceOperationCheck) {
	t.Helper()
	checks := everyRegistryCheck(eng.registry)
	result, err := eng.CheckBatch(t.Context(), &permissionports.BatchPermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    principal,
		UserID:         principal,
		OrganizationID: orgID,
		BusinessUnitID: pulid.MustNew("bu_"),
		Checks:         checks,
	})
	require.NoError(t, err)
	require.Len(t, result.Results, len(checks))
	return result, checks
}

func TestCheck_AReadOnlySupportPrincipalIsRefusedEveryNonReadOperationInTheRegistry(t *testing.T) {
	t.Parallel()

	eng, _, _, _ := setupTestEngine(t)
	eng.support = &grantedSupport{registry: eng.registry, scope: supportaccess.ScopeReadOnly, live: true}
	principal := pulid.MustNew(supportaccess.PrincipalIDPrefix)

	result, checks := checkAsSupport(t, eng, principal, pulid.MustNew("org_"))
	for i, check := range checks {
		if check.Operation != permissiondomain.OpRead {
			assert.False(t, result.Results[i].Allowed, "%s:%s", check.Resource, check.Operation)
		}
	}
}

func TestCheck_AFullSupportPrincipalIsRefusedTheDenyListAndAllowedTheRest(t *testing.T) {
	t.Parallel()

	eng, _, _, _ := setupTestEngine(t)
	eng.support = &grantedSupport{registry: eng.registry, scope: supportaccess.ScopeFull, live: true}
	principal := pulid.MustNew(supportaccess.PrincipalIDPrefix)

	effective := supportaccess.Effective(supportaccess.ScopeFull, eng.registry.All())
	result, checks := checkAsSupport(t, eng, principal, pulid.MustNew("org_"))
	for i, check := range checks {
		want := slices.Contains(effective[check.Resource], check.Operation)
		assert.Equal(t, want, result.Results[i].Allowed, "%s:%s", check.Resource, check.Operation)
		if !supportaccess.Allows(supportaccess.ScopeFull, check.Resource, check.Operation) {
			assert.False(t, result.Results[i].Allowed, "the deny list holds through a parent: %s:%s",
				check.Resource, check.Operation)
		}
	}
}

func TestCheck_ASupportPrincipalWithoutALiveGrantHoldsNothingAndNoRoleIsRead(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	eng.support = &grantedSupport{registry: eng.registry, scope: supportaccess.ScopeFull, live: false}
	principal := pulid.MustNew(supportaccess.PrincipalIDPrefix)

	result, _ := checkAsSupport(t, eng, principal, pulid.MustNew("org_"))
	for _, r := range result.Results {
		assert.False(t, r.Allowed)
	}
	roleRepo.AssertNotCalled(t, "GetUserRoleAssignments", mock.Anything, mock.Anything, mock.Anything)
	cacheRepo.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}
