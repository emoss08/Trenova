package tenantbootstrap

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fullGrants(
	registry *permissioncatalog.Registry,
	roleID pulid.ID,
) []*rbac.ResourcePermission {
	held := make([]*rbac.ResourcePermission, 0, len(registry.All()))
	for _, def := range registry.All() {
		ops := make([]permission.Operation, 0, len(def.Operations))
		for _, op := range def.Operations {
			ops = append(ops, op.Operation)
		}
		held = append(held, &rbac.ResourcePermission{
			ID:         pulid.MustNew("rp_"),
			RoleID:     roleID,
			Resource:   def.Resource,
			Operations: ops,
			DataScope:  permission.DataScopeOrganization,
		})
	}
	return held
}

func TestPlanAdminGrants_NothingToDoForACompleteRole(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	roleID := pulid.MustNew("rol_")

	plan := planAdminGrants(registry, roleID, fullGrants(registry, roleID), 100)

	assert.Empty(t, plan.add)
	assert.Empty(t, plan.extend)
}

func TestPlanAdminGrants_AddsAResourceTheRoleNeverHad(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	roleID := pulid.MustNew("rol_")
	held := fullGrants(registry, roleID)
	var dropped *rbac.ResourcePermission
	held, dropped = held[1:], held[0]

	plan := planAdminGrants(registry, roleID, held, 100)

	require.Len(t, plan.add, 1)
	assert.Equal(t, dropped.Resource, plan.add[0].Resource)
	assert.Equal(t, roleID, plan.add[0].RoleID)
	assert.ElementsMatch(
		t,
		registry.GetOperationsForResource(dropped.Resource),
		plan.add[0].Operations,
	)
	assert.Equal(t, permission.DataScopeOrganization, plan.add[0].DataScope)
	assert.Empty(t, plan.extend)
}

func TestPlanAdminGrants_ExtendsAResourceMissingAnOperation(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	roleID := pulid.MustNew("rol_")
	held := fullGrants(registry, roleID)
	var target *rbac.ResourcePermission
	for _, rp := range held {
		if len(rp.Operations) > 1 {
			target = rp
			break
		}
	}
	require.NotNil(t, target)
	missing := target.Operations[len(target.Operations)-1]
	target.Operations = target.Operations[:len(target.Operations)-1]

	plan := planAdminGrants(registry, roleID, held, 100)

	assert.Empty(t, plan.add)
	require.Len(t, plan.extend, 1)
	assert.Equal(t, target.ID, plan.extend[0].ID)
	assert.Contains(t, plan.extend[0].Operations, missing)
	assert.Equal(t, int64(100), plan.extend[0].UpdatedAt)
}

func TestPlanAdminGrants_KeepsOperationsTheRegistryNoLongerDeclares(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	roleID := pulid.MustNew("rol_")
	held := fullGrants(registry, roleID)
	held[0].Operations = append(held[0].Operations, permission.OpManage)
	registered := len(registry.GetOperationsForResource(held[0].Resource))

	plan := planAdminGrants(registry, roleID, held, 100)

	if registered > 0 {
		assert.Empty(t, plan.extend, "a grant beyond the registry is left for a person to review")
	}
}

func TestPlanAdminGrants_GrantsLeaveToAnAdministratorWhoHoldsOnlyWorkers(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	roleID := pulid.MustNew("rol_")
	held := make([]*rbac.ResourcePermission, 0, len(registry.All()))
	for _, rp := range fullGrants(registry, roleID) {
		if rp.Resource != permission.ResourceWorkerLeave.String() {
			held = append(held, rp)
		}
	}

	plan := planAdminGrants(registry, roleID, held, 100)

	require.Len(
		t,
		plan.add,
		1,
		"worker_leave is its own grant, so a worker grant does not cover it",
	)
	assert.Equal(t, permission.ResourceWorkerLeave.String(), plan.add[0].Resource)
	assert.ElementsMatch(t, []permission.Operation{
		permission.OpRead,
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpApprove,
		permission.OpManage,
		permission.OpExport,
	}, plan.add[0].Operations)
	assert.Equal(t, permission.DataScopeOrganization, plan.add[0].DataScope)
	assert.Empty(t, plan.extend)
}

func TestPlanAdminGrants_GrantsEveryHRKindToAnAdministratorWhoHoldsOnlyWorkers(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	roleID := pulid.MustNew("rol_")
	own := []permission.Resource{
		permission.ResourceWorkerInjury,
		permission.ResourceWorkerDOTTest,
		permission.ResourceBenefitPlan,
		permission.ResourceWorkerEmploymentEvent,
		permission.ResourceWorkerDisciplinaryAction,
		permission.ResourcePerformanceReview,
		permission.ResourceTimesheet,
		permission.ResourceProfileChangeRequest,
	}
	held := make([]*rbac.ResourcePermission, 0, len(registry.All()))
	for _, rp := range fullGrants(registry, roleID) {
		if !slices.Contains(own, permission.Resource(rp.Resource)) {
			held = append(held, rp)
		}
	}

	plan := planAdminGrants(registry, roleID, held, 100)

	require.Len(t, plan.add, len(own), "each is its own grant, so a worker grant does not cover it")
	for _, add := range plan.add {
		assert.Contains(t, own, permission.Resource(add.Resource))
		assert.ElementsMatch(t, registry.GetOperationsForResource(add.Resource), add.Operations,
			add.Resource)
		assert.Equal(t, permission.DataScopeOrganization, add.DataScope)
	}
	assert.Empty(t, plan.extend)
}

func TestPlanAdminGrants_GivesAnAdministratorPortalInvitationsAndKeepsRetiredOperations(
	t *testing.T,
) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	roleID := pulid.MustNew("rol_")
	portal := permission.ResourceDriverPortal.String()
	held := make([]*rbac.ResourcePermission, 0, len(registry.All()))
	for _, rp := range fullGrants(registry, roleID) {
		if rp.Resource != portal {
			held = append(held, rp)
		}
	}

	plan := planAdminGrants(registry, roleID, held, 100)
	require.Len(
		t,
		plan.add,
		1,
		"driver_portal is its own grant, so a worker grant does not cover it",
	)
	assert.Equal(t, portal, plan.add[0].Resource)
	assert.ElementsMatch(t, []permission.Operation{
		permission.OpRead,
		permission.OpCreate,
		permission.OpCancel,
	}, plan.add[0].Operations, "inviting and revoking are the portal's own operations")

	previous := &rbac.ResourcePermission{
		ID:       pulid.MustNew("rp_"),
		RoleID:   roleID,
		Resource: portal,
		Operations: []permission.Operation{
			permission.OpRead, permission.OpCreate, permission.OpUpdate,
			permission.OpExport, permission.OpImport, permission.OpCancel,
		},
		DataScope: permission.DataScopeOrganization,
	}
	plan = planAdminGrants(registry, roleID, append(held, previous), 100)
	assert.Empty(t, plan.add)
	assert.Empty(
		t,
		plan.extend,
		"an administrator who held the old operations already invites and revokes",
	)
}
