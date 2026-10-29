package pageaccessservice_test

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/pageaccessservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pageaccess"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const fixture = `{
  "version": "sha256:test",
  "screens": [
    {"path": "/shipments", "name": "Shipments",
     "requires": [{"resource": "shipment", "operation": "read"}],
     "operations": ["ShipmentList"],
     "endpoints": [{"method": "POST", "path": "/shipments/", "requires": [{"resource": "shipment", "operation": "create"}]}],
     "reads": ["shipment"], "references": ["worker"], "declared": ["customer"]},
    {"path": "/invoices/:id", "name": "", "requires": [], "operations": ["InvoiceDetail"],
     "endpoints": [], "reads": ["invoice"], "references": ["shipment"], "declared": []}
  ],
  "operations": [
    {"name": "ShipmentList", "type": "query", "reads": ["shipment"],
     "linked": [
       {"kind": "customer", "fields": ["creditLimit", "name"]},
       {"kind": "worker", "fields": ["phoneNumber", "wholeName"]}
     ], "declared": ["customer"]},
    {"name": "InvoiceDetail", "type": "query", "reads": ["invoice"],
     "linked": [{"kind": "shipment", "fields": ["ShipmentMove.status", "proNumber"]}],
     "declared": []}
  ]
}`

// engine allows every check but those on Denied resources, answers the
// person's sensitivity as Max, and counts batches.
type engine struct {
	mocks.AllowAllPermissionEngine

	Denied  []string
	Max     permission.FieldSensitivity
	batches int
}

func (e *engine) CheckBatch(
	_ context.Context,
	req *permissionports.BatchPermissionCheckRequest,
) (*permissionports.BatchPermissionCheckResult, error) {
	e.batches++
	out := &permissionports.BatchPermissionCheckResult{}
	for _, check := range req.Checks {
		out.Results = append(out.Results, permissionports.PermissionCheckResult{
			Allowed: !slices.Contains(e.Denied, check.Resource),
		})
	}
	return out, nil
}

func (e *engine) GetResourcePermissions(
	context.Context,
	pulid.ID,
	pulid.ID,
	string,
) (*permissionports.ResourcePermissionDetail, error) {
	return &permissionports.ResourcePermissionDetail{MaxSensitivity: e.Max}, nil
}

func newService(
	t *testing.T,
	eng permissionports.PermissionEngine,
	roles permissionports.RoleRepository,
) *pageaccessservice.Service {
	t.Helper()
	manifest, err := pageaccess.Load([]byte(fixture))
	require.NoError(t, err)
	return pageaccessservice.NewWithManifest(pageaccessservice.Params{
		Engine:   eng,
		Roles:    roles,
		Registry: permissioncatalog.NewRegistry(),
	}, manifest)
}

var tenant = pagination.TenantInfo{
	OrgID:  pulid.MustNew("org_"),
	BuID:   pulid.MustNew("bu_"),
	UserID: pulid.MustNew("usr_"),
}

func grant(resource string, ops ...permission.Operation) pageaccessservice.DraftPermission {
	return pageaccessservice.DraftPermission{Resource: resource, Operations: ops}
}

func kind(t *testing.T, access *pageaccessservice.Access, resource string) pageaccessservice.Kind {
	t.Helper()
	for _, k := range access.Kinds {
		if k.Resource == resource {
			return k
		}
	}
	require.Failf(t, "kind not described", "%s", resource)
	return pageaccessservice.Kind{}
}

func decisions(k pageaccessservice.Kind) map[string]pageaccessservice.Decision {
	out := make(map[string]pageaccessservice.Decision, len(k.Fields))
	for _, field := range k.Fields {
		out[field.Name] = field.Decision
	}
	return out
}

func TestPreview_ADraftRoleSeesALinkedKindThroughItsViewOnly(t *testing.T) {
	t.Parallel()

	svc := newService(t, &engine{}, permissionportsmock.NewMockRoleRepository(t))
	access, err := svc.Preview(t.Context(), &pageaccessservice.PreviewRequest{
		Tenant:   tenant,
		Location: "/shipments",
		Draft: &pageaccessservice.RoleDraft{
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []pageaccessservice.DraftPermission{
				grant("shipment", permission.OpRead),
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, access)

	assert.Equal(t, "/shipments", access.Path)
	assert.False(t, access.Blocked)
	assert.Equal(t, []pageaccessservice.Requirement{{
		Resource: "shipment", ResourceName: "Shipment", Operation: "read",
		OperationName: "View", Allowed: true,
	}}, access.Requires)
	assert.False(t, access.Endpoints[0].Requires[0].Allowed, "the draft holds no shipment create")

	shipment := kind(t, access, "shipment")
	assert.Equal(t, pageaccess.UseRequired, shipment.Use)
	assert.True(t, shipment.CanRead)

	customer := kind(t, access, "customer")
	assert.Equal(t, pageaccess.UseDeclared, customer.Use)
	assert.False(t, customer.CanRead)
	assert.Equal(t, map[string]pageaccessservice.Decision{
		"creditLimit": pageaccessservice.DecisionWithheldKind,
		"name":        pageaccessservice.DecisionVisible,
	}, decisions(customer))
	assert.Equal(t, 1, customer.Withheld)

	worker := kind(t, access, "worker")
	assert.Equal(t, pageaccess.UseReference, worker.Use)
	assert.Equal(t, map[string]pageaccessservice.Decision{
		"phoneNumber": pageaccessservice.DecisionWithheldKind,
		"wholeName":   pageaccessservice.DecisionVisible,
	}, decisions(worker))
	assert.Equal(t, 2, access.Withheld)

	var order []pageaccess.Use
	for _, k := range access.Kinds {
		order = append(order, k.Use)
	}
	assert.Equal(t, []pageaccess.Use{
		pageaccess.UseRequired, pageaccess.UseDeclared, pageaccess.UseReference,
	}, order, "kinds come strongest use first")
}

func TestPreview_TheKindsReadShowsTheWholeLinkedRecord(t *testing.T) {
	t.Parallel()

	svc := newService(t, &engine{}, permissionportsmock.NewMockRoleRepository(t))
	access, err := svc.Preview(t.Context(), &pageaccessservice.PreviewRequest{
		Tenant:   tenant,
		Location: "/shipments",
		Draft: &pageaccessservice.RoleDraft{
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []pageaccessservice.DraftPermission{
				grant("shipment", permission.OpRead),
				grant("customer", permission.OpUpdate),
			},
		},
	})
	require.NoError(t, err)

	customer := kind(t, access, "customer")
	assert.True(t, customer.CanRead, "update implies read")
	assert.Equal(t, 0, customer.Withheld)
}

func TestPreview_AViewFieldAboveTheRolesCeilingIsWithheld(t *testing.T) {
	t.Parallel()

	svc := newService(t, &engine{}, permissionportsmock.NewMockRoleRepository(t))
	access, err := svc.Preview(t.Context(), &pageaccessservice.PreviewRequest{
		Tenant:   tenant,
		Location: "/shipments",
		Draft: &pageaccessservice.RoleDraft{
			MaxSensitivity: permission.SensitivityPublic,
			Permissions: []pageaccessservice.DraftPermission{
				grant("shipment", permission.OpRead),
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, pageaccessservice.DecisionWithheldSensitivity,
		decisions(kind(t, access, "worker"))["wholeName"])
}

func TestPreview_ARoleWithoutTheScreensReadIsBlocked(t *testing.T) {
	t.Parallel()

	svc := newService(t, &engine{}, permissionportsmock.NewMockRoleRepository(t))
	access, err := svc.Preview(t.Context(), &pageaccessservice.PreviewRequest{
		Tenant:   tenant,
		Location: "/invoices/inv_1",
		Draft:    &pageaccessservice.RoleDraft{MaxSensitivity: permission.SensitivityInternal},
	})
	require.NoError(t, err)

	assert.Equal(t, "/invoices/:id", access.Path)
	assert.True(t, access.Blocked, "the screen reads invoices as roots")
	assert.Equal(t, map[string]pageaccessservice.Decision{
		"ShipmentMove.status": pageaccessservice.DecisionWithheldKind,
		"proNumber":           pageaccessservice.DecisionVisible,
	}, decisions(kind(t, access, "shipment")),
		"a part of a linked shipment has no view of its own")
}

func TestPreview_ASavedRoleIncludesTheRolesItInherits(t *testing.T) {
	t.Parallel()

	parent := &rbac.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: tenant.OrgID,
		MaxSensitivity: permission.SensitivityInternal,
		Permissions: []*rbac.ResourcePermission{
			{Resource: "customer", Operations: []permission.Operation{permission.OpRead}},
		},
	}
	role := &rbac.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: tenant.OrgID,
		MaxSensitivity: permission.SensitivityPublic,
		ParentRoleIDs:  []pulid.ID{parent.ID},
		Permissions: []*rbac.ResourcePermission{
			{Resource: "shipment", Operations: []permission.Operation{permission.OpRead}},
		},
	}
	roles := permissionportsmock.NewMockRoleRepository(t)
	roles.EXPECT().
		GetByID(mock.Anything, permissionports.GetRoleByIDRequest{ID: role.ID, TenantInfo: tenant}).
		Return(role, nil)
	roles.EXPECT().
		GetRolesWithInheritance(mock.Anything, []pulid.ID{parent.ID}).
		Return([]*rbac.Role{parent}, nil)

	svc := newService(t, &engine{}, roles)
	access, err := svc.Preview(t.Context(), &pageaccessservice.PreviewRequest{
		Tenant:   tenant,
		Location: "/shipments",
		RoleID:   role.ID,
	})
	require.NoError(t, err)

	assert.True(t, kind(t, access, "customer").CanRead)
	assert.Equal(t, pageaccessservice.DecisionVisible,
		decisions(kind(t, access, "worker"))["wholeName"],
		"the inherited role's ceiling is the wider one")
}

func TestPreview_RefusesARoleInheritingFromAnotherOrganization(t *testing.T) {
	t.Parallel()

	parent := &rbac.Role{ID: pulid.MustNew("rol_"), OrganizationID: pulid.MustNew("org_")}
	roles := permissionportsmock.NewMockRoleRepository(t)
	roles.EXPECT().
		GetRolesWithInheritance(mock.Anything, []pulid.ID{parent.ID}).
		Return([]*rbac.Role{parent}, nil)

	svc := newService(t, &engine{}, roles)
	_, err := svc.Preview(t.Context(), &pageaccessservice.PreviewRequest{
		Tenant:   tenant,
		Location: "/shipments",
		Draft: &pageaccessservice.RoleDraft{
			MaxSensitivity: permission.SensitivityInternal,
			ParentRoleIDs:  []pulid.ID{parent.ID},
		},
	})

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "parentRoleIds", validation.Field)
}

func TestPreview_RefusesAnAmbiguousOrInvalidRole(t *testing.T) {
	t.Parallel()

	svc := newService(t, &engine{}, permissionportsmock.NewMockRoleRepository(t))
	cases := map[string]*pageaccessservice.PreviewRequest{
		"neither": {Tenant: tenant, Location: "/shipments"},
		"both": {
			Tenant: tenant, Location: "/shipments", RoleID: pulid.MustNew("rol_"),
			Draft: &pageaccessservice.RoleDraft{MaxSensitivity: permission.SensitivityInternal},
		},
		"unknown resource": {
			Tenant: tenant, Location: "/shipments",
			Draft: &pageaccessservice.RoleDraft{
				MaxSensitivity: permission.SensitivityInternal,
				Permissions: []pageaccessservice.DraftPermission{
					grant("nope", permission.OpRead),
				},
			},
		},
		"undeclared operation": {
			Tenant: tenant, Location: "/shipments",
			Draft: &pageaccessservice.RoleDraft{
				MaxSensitivity: permission.SensitivityInternal,
				Permissions: []pageaccessservice.DraftPermission{
					grant("customer", permission.Operation("teleport")),
				},
			},
		},
		"bad sensitivity": {
			Tenant: tenant, Location: "/shipments",
			Draft: &pageaccessservice.RoleDraft{MaxSensitivity: "secret"},
		},
		"no location": {
			Tenant: tenant,
			Draft:  &pageaccessservice.RoleDraft{MaxSensitivity: permission.SensitivityInternal},
		},
	}
	for name, req := range cases {
		_, err := svc.Preview(t.Context(), req)
		assert.Error(t, err, name)
	}
}

func TestPreview_NoScreenAtTheLocationIsNil(t *testing.T) {
	t.Parallel()

	svc := newService(t, &engine{}, permissionportsmock.NewMockRoleRepository(t))
	access, err := svc.Preview(t.Context(), &pageaccessservice.PreviewRequest{
		Tenant:   tenant,
		Location: "/nowhere",
		Draft:    &pageaccessservice.RoleDraft{MaxSensitivity: permission.SensitivityInternal},
	})
	require.NoError(t, err)
	assert.Nil(t, access)
}

func TestForCaller_AsksTheEngineOnceForEveryCheckTheScreenNeeds(t *testing.T) {
	t.Parallel()

	eng := &engine{Denied: []string{"customer"}, Max: permission.SensitivityInternal}
	svc := newService(t, eng, permissionportsmock.NewMockRoleRepository(t))
	access, err := svc.ForCaller(t.Context(), permissionports.UserActor(tenant), "/shipments/")
	require.NoError(t, err)

	assert.Equal(t, 1, eng.batches)
	assert.False(t, access.Blocked)
	assert.True(t, access.Endpoints[0].Requires[0].Allowed)
	customer := kind(t, access, "customer")
	assert.False(t, customer.CanRead)
	assert.Equal(t, pageaccessservice.DecisionWithheldKind, decisions(customer)["creditLimit"])
	assert.True(t, kind(t, access, "worker").CanRead)
	assert.Equal(t, 1, access.Withheld)
}

func TestScreens_ListsEveryScreen(t *testing.T) {
	t.Parallel()

	svc := newService(t, &engine{}, permissionportsmock.NewMockRoleRepository(t))
	assert.Equal(t, []pageaccessservice.Screen{
		{Path: "/shipments", Name: "Shipments"},
		{Path: "/invoices/:id", Name: ""},
	}, svc.Screens())
}
