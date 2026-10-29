//go:build integration

package pageaccessservice_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/rolerepository"
	"github.com/emoss08/trenova/internal/modules/identity/app/pageaccessservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func insertRole(
	t *testing.T,
	db *keel.DB,
	tenant *seedtest.TestData,
	maxSensitivity permission.FieldSensitivity,
	parents []pulid.ID,
	grants ...*rbac.ResourcePermission,
) *rbac.Role {
	t.Helper()

	ctx := tenant.Ctx(t.Context())
	role := &rbac.Role{
		Name:           "Preview " + pulid.MustNew("rol_").String(),
		MaxSensitivity: maxSensitivity,
		ParentRoleIDs:  parents,
	}
	require.NoError(t, permissiondb.Roles.Create(ctx, db, role))
	for _, rp := range grants {
		rp.RoleID = role.ID
		rp.DataScope = permission.DataScopeOrganization
	}
	if len(grants) > 0 {
		require.NoError(t, permissiondb.ResourcePermissions.CreateMany(ctx, db, grants))
	}
	return role
}

func TestPreview_ReadsASavedRoleAndItsAncestorsFromItsOwnOrganization(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "PA")

	parent := insertRole(t, db, tenantA, permission.SensitivityInternal, nil,
		&rbac.ResourcePermission{
			Resource:   permission.ResourceCustomer.String(),
			Operations: []permission.Operation{permission.OpRead},
		})
	child := insertRole(t, db, tenantA, permission.SensitivityPublic, []pulid.ID{parent.ID},
		&rbac.ResourcePermission{
			Resource:   permission.ResourceShipment.String(),
			Operations: []permission.Operation{permission.OpRead},
		})
	other := insertRole(t, db, tenantB, permission.SensitivityRestricted, nil)

	repo := rolerepository.New(rolerepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	svc := newService(t, &engine{}, repo)
	ctx = tenantA.Ctx(ctx)
	tenant := pagination.TenantInfo{
		OrgID:  tenantA.Organization.ID,
		BuID:   tenantA.BusinessUnit.ID,
		UserID: tenantA.User.ID,
	}

	t.Run("a saved role shows what it and the roles it inherits grant", func(t *testing.T) {
		access, err := svc.Preview(ctx, &pageaccessservice.PreviewRequest{
			Tenant:   tenant,
			Location: "/shipments",
			RoleID:   child.ID,
		})
		require.NoError(t, err)
		require.NotNil(t, access)

		assert.False(t, access.Blocked)
		assert.True(t, kind(t, access, "customer").CanRead, "inherited from the parent")
		assert.Equal(t, 0, kind(t, access, "customer").Withheld)
		worker := kind(t, access, "worker")
		assert.False(t, worker.CanRead)
		assert.Equal(t, map[string]pageaccessservice.Decision{
			"phoneNumber": pageaccessservice.DecisionWithheldKind,
			"wholeName":   pageaccessservice.DecisionVisible,
		}, decisions(worker), "the parent's Internal ceiling is the wider one")
	})

	t.Run("another organization's role is not found", func(t *testing.T) {
		_, err := svc.Preview(ctx, &pageaccessservice.PreviewRequest{
			Tenant:   tenant,
			Location: "/shipments",
			RoleID:   other.ID,
		})
		require.Error(t, err)
		assert.True(t, errortypes.IsNotFoundError(err))
	})

	t.Run("a draft cannot inherit another organization's role", func(t *testing.T) {
		_, err := svc.Preview(ctx, &pageaccessservice.PreviewRequest{
			Tenant:   tenant,
			Location: "/shipments",
			Draft: &pageaccessservice.RoleDraft{
				MaxSensitivity: permission.SensitivityInternal,
				ParentRoleIDs:  []pulid.ID{other.ID},
			},
		})
		require.Error(t, err)
		assert.False(t, errortypes.IsNotFoundError(err))
	})
}
