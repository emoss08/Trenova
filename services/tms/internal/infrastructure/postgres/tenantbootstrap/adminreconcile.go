package tenantbootstrap

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac/rbacdb"
	"github.com/emoss08/trenova/shared/pulid"
)

type AdminReconcileResult struct {
	RolesChecked   int
	GrantsAdded    int
	GrantsExtended int
}

type adminGrantPlan struct {
	add    []*rbac.ResourcePermission
	extend []*rbac.ResourcePermission
}

func ReconcileAdminRole(
	ctx context.Context,
	db *keel.DB,
	scope tenantscope.Scope,
	registry *permissioncatalog.Registry,
) (AdminReconcileResult, error) {
	if registry == nil {
		registry = permissioncatalog.NewRegistry()
	}

	r := rbacdb.Roles
	roles, err := tenantscope.Scoped(ctx, r.Pick(r.ID).Where(
		r.OrganizationID.Eq(scope.OrganizationID),
		r.BusinessUnitID.Eq(scope.BusinessUnitID),
		r.IsSystem.IsTrue(),
		r.Name.Eq(AdministratorRoleName),
	)).All(ctx, db)
	if err != nil {
		return AdminReconcileResult{}, fmt.Errorf("list administrator roles: %w", err)
	}

	result := AdminReconcileResult{RolesChecked: len(roles)}
	if len(roles) == 0 {
		return result, nil
	}

	roleIDs := make([]pulid.ID, 0, len(roles))
	for _, role := range roles {
		roleIDs = append(roleIDs, role.ID)
	}

	rp := rbacdb.ResourcePermissions
	existing, err := rp.Where(rp.RoleID.In(roleIDs...)).All(ctx, db)
	if err != nil {
		return result, fmt.Errorf("list administrator grants: %w", err)
	}

	byRole := make(map[pulid.ID][]*rbac.ResourcePermission, len(roleIDs))
	for _, held := range existing {
		byRole[held.RoleID] = append(byRole[held.RoleID], held)
	}

	now := scope.NowUnix()
	for _, roleID := range roleIDs {
		plan := planAdminGrants(registry, roleID, byRole[roleID], now)
		if err = rp.CreateMany(ctx, db, plan.add); err != nil {
			return result, fmt.Errorf("grant new resources to role %s: %w", roleID, err)
		}
		for _, held := range plan.extend {
			if err = rp.Save(ctx, db, held, rp.Operations); err != nil {
				return result, fmt.Errorf("grant new operations to role %s: %w", roleID, err)
			}
		}
		result.GrantsAdded += len(plan.add)
		result.GrantsExtended += len(plan.extend)
	}

	return result, nil
}

func planAdminGrants(
	registry *permissioncatalog.Registry,
	roleID pulid.ID,
	held []*rbac.ResourcePermission,
	now int64,
) adminGrantPlan {
	byResource := make(map[string]*rbac.ResourcePermission, len(held))
	for _, rp := range held {
		byResource[rp.Resource] = rp
	}

	definitions := registry.All()
	slices.SortFunc(definitions, func(a, b *permissionspec.ResourceDefinition) int {
		if a.Resource < b.Resource {
			return -1
		}
		if a.Resource > b.Resource {
			return 1
		}
		return 0
	})

	var plan adminGrantPlan
	for _, def := range definitions {
		declared := make([]permission.Operation, 0, len(def.Operations))
		for _, op := range def.Operations {
			declared = append(declared, op.Operation)
		}

		rp, ok := byResource[def.Resource]
		if !ok {
			plan.add = append(plan.add, &rbac.ResourcePermission{
				ID:         pulid.MustNew("rp_"),
				RoleID:     roleID,
				Resource:   def.Resource,
				Operations: declared,
				DataScope:  permission.DataScopeOrganization,
				CreatedAt:  now,
				UpdatedAt:  now,
			})
			continue
		}

		holds := permission.NewOperationSet(rp.Operations...)
		missing := false
		for _, op := range declared {
			if !holds.Has(op) {
				missing = true
				break
			}
		}
		if !missing {
			continue
		}

		extended := slices.Clone(rp.Operations)
		for _, op := range declared {
			if !holds.Has(op) {
				extended = append(extended, op)
			}
		}
		rp.Operations = extended
		rp.UpdatedAt = now
		plan.extend = append(plan.extend, rp)
	}

	return plan
}
