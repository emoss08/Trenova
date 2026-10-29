package domainregistry

import (
	"reflect"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
)

func TestRegisterEntitiesIncludesRBACModels(t *testing.T) {
	t.Parallel()

	registered := make(map[reflect.Type]struct{}, len(RegisterEntities()))
	for _, entity := range RegisterEntities() {
		registered[reflect.TypeOf(entity)] = struct{}{}
	}

	required := []any{
		&rbac.Role{},
		&rbac.ResourcePermission{},
		&rbac.UserRoleAssignment{},
		&rbac.RoleHierarchyEdge{},
		&rbac.RoleConstraint{},
		&rbac.RoleConstraintRole{},
	}
	for _, entity := range required {
		if _, ok := registered[reflect.TypeOf(entity)]; !ok {
			t.Fatalf("expected %T to be registered", entity)
		}
	}
}
