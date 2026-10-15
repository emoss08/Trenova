package tenantbootstrap

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fullGrants(registry *permission.Registry, roleID pulid.ID) []*permission.ResourcePermission {
	held := make([]*permission.ResourcePermission, 0, len(registry.All()))
	for _, def := range registry.All() {
		ops := make([]permission.Operation, 0, len(def.Operations))
		for _, op := range def.Operations {
			ops = append(ops, op.Operation)
		}
		held = append(held, &permission.ResourcePermission{
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

	registry := permission.NewRegistry()
	roleID := pulid.MustNew("rol_")

	plan := planAdminGrants(registry, roleID, fullGrants(registry, roleID), 100)

	assert.Empty(t, plan.add)
	assert.Empty(t, plan.extend)
}

func TestPlanAdminGrants_AddsAResourceTheRoleNeverHad(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	roleID := pulid.MustNew("rol_")
	held := fullGrants(registry, roleID)
	var dropped *permission.ResourcePermission
	held, dropped = held[1:], held[0]

	plan := planAdminGrants(registry, roleID, held, 100)

	require.Len(t, plan.add, 1)
	assert.Equal(t, dropped.Resource, plan.add[0].Resource)
	assert.Equal(t, roleID, plan.add[0].RoleID)
	assert.ElementsMatch(t, registry.GetOperationsForResource(dropped.Resource), plan.add[0].Operations)
	assert.Equal(t, permission.DataScopeOrganization, plan.add[0].DataScope)
	assert.Empty(t, plan.extend)
}

func TestPlanAdminGrants_ExtendsAResourceMissingAnOperation(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	roleID := pulid.MustNew("rol_")
	held := fullGrants(registry, roleID)
	var target *permission.ResourcePermission
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

	registry := permission.NewRegistry()
	roleID := pulid.MustNew("rol_")
	held := fullGrants(registry, roleID)
	held[0].Operations = append(held[0].Operations, permission.OpManage)
	registered := len(registry.GetOperationsForResource(held[0].Resource))

	plan := planAdminGrants(registry, roleID, held, 100)

	if registered > 0 {
		assert.Empty(t, plan.extend, "a grant beyond the registry is left for a person to review")
	}
}
