package domainregistry

import (
	"reflect"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func TestRegisterEntitiesIncludesRBACModels(t *testing.T) {
	t.Parallel()

	registered := make(map[reflect.Type]struct{}, len(RegisterEntities()))
	for _, entity := range RegisterEntities() {
		registered[reflect.TypeOf(entity)] = struct{}{}
	}

	required := []any{
		&permission.Role{},
		&permission.ResourcePermission{},
		&permission.UserRoleAssignment{},
		&permission.RoleHierarchyEdge{},
		&permission.RoleConstraint{},
		&permission.RoleConstraintRole{},
	}
	for _, entity := range required {
		if _, ok := registered[reflect.TypeOf(entity)]; !ok {
			t.Fatalf("expected %T to be registered", entity)
		}
	}
}
