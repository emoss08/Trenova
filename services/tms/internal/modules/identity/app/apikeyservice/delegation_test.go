package apikeyservice

import (
	"testing"

	permissiondomain "github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/apikeyports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func actorHolding(
	t *testing.T,
	resource permissiondomain.Resource,
	ops ...permissiondomain.Operation,
) *permissionportsmock.MockPermissionEngine {
	t.Helper()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.EXPECT().GetEffectivePermissions(mock.Anything, mock.Anything, mock.Anything).
		Return(&permissionports.EffectivePermissions{
			Resources: map[string]permissionports.EffectiveResourcePermission{
				resource.String(): {
					Operations: ops,
					DataScope:  permissiondomain.DataScopeOrganization,
				},
			},
		}, nil).Once()
	return engine
}

func TestCreateAPIKey_RefusesGrantsBeyondTheCreator(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	svc.engine = actorHolding(t, permissiondomain.ResourceCustomer, permissiondomain.OpRead)
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	userID := pulid.MustNew("usr_")
	repo.EXPECT().CountActiveByCreator(mock.Anything, tenantInfo, userID).Return(0, nil)

	_, err := svc.CreateAPIKey(t.Context(), tenantInfo, &apikeyports.CreateAPIKeyRequest{
		Name: "Customer sync",
		Permissions: []apikeyports.APIKeyPermissionInput{{
			Resource: permissiondomain.ResourceCustomer.String(),
			Operations: []permissiondomain.Operation{
				permissiondomain.OpRead,
				permissiondomain.OpUpdate,
			},
			DataScope: permissiondomain.DataScopeOrganization,
		}},
	}, userID)

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	repo.AssertNotCalled(t, "CreateWithPermissions", mock.Anything, mock.Anything, mock.Anything)
}

func TestCreateAPIKey_RefusesAWiderScopeThanTheCreatorHolds(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	svc.engine = actorHolding(t, permissiondomain.ResourceCustomer, permissiondomain.OpRead)
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	userID := pulid.MustNew("usr_")
	repo.EXPECT().CountActiveByCreator(mock.Anything, tenantInfo, userID).Return(0, nil)

	_, err := svc.CreateAPIKey(t.Context(), tenantInfo, &apikeyports.CreateAPIKeyRequest{
		Name: "Customer sync",
		Permissions: []apikeyports.APIKeyPermissionInput{{
			Resource:   permissiondomain.ResourceCustomer.String(),
			Operations: []permissiondomain.Operation{permissiondomain.OpRead},
			DataScope:  permissiondomain.DataScopeAll,
		}},
	}, userID)

	require.Error(t, err)
}

func TestUpdateAPIKey_RefusesGrantsBeyondTheEditor(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	svc.engine = actorHolding(t, permissiondomain.ResourceCustomer, permissiondomain.OpRead)
	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}

	_, err := svc.UpdateAPIKey(t.Context(), tenantInfo, pulid.MustNew("ak_"), &apikeyports.UpdateAPIKeyRequest{
		Name: "Customer sync",
		Permissions: []apikeyports.APIKeyPermissionInput{{
			Resource:   permissiondomain.ResourceLocation.String(),
			Operations: []permissiondomain.Operation{permissiondomain.OpRead},
			DataScope:  permissiondomain.DataScopeOrganization,
		}},
	})

	require.Error(t, err)
	repo.AssertNotCalled(t, "UpdateWithPermissions", mock.Anything, mock.Anything, mock.Anything)
}
