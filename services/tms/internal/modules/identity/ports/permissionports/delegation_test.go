package permissionports

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
)

func holding(ops ...permission.Operation) *EffectivePermissions {
	return &EffectivePermissions{
		Resources: map[string]EffectiveResourcePermission{
			"invoice": {Operations: ops, DataScope: permission.DataScopeOrganization},
		},
	}
}

func TestDelegationViolation_AGrantCarriesWhatItImplies(t *testing.T) {
	t.Parallel()

	violation, refused := holding(permission.OpApprove).DelegationViolation(
		"invoice",
		[]permission.Operation{permission.OpUpdate},
		permission.DataScopeOrganization,
	)

	assert.False(t, refused, "holding approve means holding the update it implies: %+v", violation)
}

func TestDelegationViolation_RefusesWhatAGrantWouldImplyBeyondTheActor(t *testing.T) {
	t.Parallel()

	violation, refused := holding(permission.OpManage).DelegationViolation(
		"invoice",
		[]permission.Operation{permission.OpImport},
		permission.DataScopeOrganization,
	)

	assert.True(t, refused)
	assert.Equal(t, DelegationOperationNotHeld, violation.Kind)
	assert.Contains(
		t,
		[]permission.Operation{permission.OpCreate, permission.OpImport},
		violation.Operation,
	)
}
