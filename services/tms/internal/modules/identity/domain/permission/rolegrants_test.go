package permission_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
)

func TestGrantsOf_UnitesRolesWithTheirImpliedOperations(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	grants := permission.GrantsOf(
		&permission.Role{
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []*permission.ResourcePermission{
				{Resource: "customer", Operations: []permission.Operation{permission.OpUpdate}},
			},
		},
		&permission.Role{
			MaxSensitivity: permission.SensitivityConfidential,
			Permissions: []*permission.ResourcePermission{
				{Resource: "worker", Operations: []permission.Operation{permission.OpRead}},
			},
		},
		nil,
	)

	assert.Equal(t, permission.SensitivityConfidential, grants.MaxSensitivity)
	assert.True(t, grants.Allows(registry, "customer", permission.OpRead))
	assert.True(t, grants.Allows(registry, "worker", permission.OpRead))
	assert.False(t, grants.Allows(registry, "worker", permission.OpUpdate))
	assert.False(t, grants.Allows(registry, "invoice", permission.OpRead))
}

func TestGrantsOf_NoRolesGrantsNothingAtPublic(t *testing.T) {
	t.Parallel()

	grants := permission.GrantsOf()
	assert.Equal(t, permission.SensitivityPublic, grants.MaxSensitivity)
	assert.False(t, grants.Allows(nil, "customer", permission.OpRead))
}

func TestRoleGrants_AParentsGrantReachesItsChildren(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	var child, parent string
	for _, def := range registry.All() {
		if def.ParentResource != "" {
			child, parent = def.Resource, def.ParentResource
			break
		}
	}
	if child == "" {
		t.Skip("the registry declares no child resource")
	}

	grants := permission.GrantsOf(&permission.Role{Permissions: []*permission.ResourcePermission{
		{Resource: parent, Operations: []permission.Operation{permission.OpRead}},
	}})
	assert.True(t, grants.Allows(registry, child, permission.OpRead))
}

func TestRoleGrants_AWorkerGrantReachesNoKindGrantedOnItsOwn(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	worker := permission.ResourceWorker.String()
	grants := permission.GrantsOf(&permission.Role{Permissions: []*permission.ResourcePermission{
		{Resource: worker, Operations: registry.GetOperationsForResource(worker)},
	}})

	for _, kind := range []permission.Resource{
		permission.ResourceWorkerInjury,
		permission.ResourceWorkerDOTTest,
		permission.ResourceBenefitPlan,
		permission.ResourcePerformanceReview,
		permission.ResourceTimesheet,
		permission.ResourceDriverPortal,
	} {
		assert.False(t, grants.Allows(registry, kind.String(), permission.OpRead),
			"a worker grant says nothing about %s", kind)
	}
	assert.True(t, grants.Allows(registry, permission.ResourceWorkerCredential.String(),
		permission.OpRead), "the rest of the worker's family follows the worker grant")
}
