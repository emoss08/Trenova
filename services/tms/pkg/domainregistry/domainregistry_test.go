package domainregistry

import (
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
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

func TestRegisterEntitiesCanBeRegisteredByBun(t *testing.T) {
	t.Parallel()

	sqlDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sql mock: %v", err)
	}
	db := bun.NewDB(sqlDB, pgdialect.New())
	defer db.Close()

	db.RegisterModel(RegisterManyToManyEntities()...)
	db.RegisterModel(RegisterEntities()...)
}
