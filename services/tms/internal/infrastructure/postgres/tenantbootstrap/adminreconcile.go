package tenantbootstrap

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/buncolgen/permissioncol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type AdminReconcileResult struct {
	RolesChecked   int
	GrantsAdded    int
	GrantsExtended int
}

type adminGrantPlan struct {
	add    []*permission.ResourcePermission
	extend []*permission.ResourcePermission
}

func ReconcileAdminRoles(
	ctx context.Context,
	db bun.IDB,
	registry *permission.Registry,
) (*AdminReconcileResult, error) {
	if registry == nil {
		registry = permission.NewRegistry()
	}

	roleCols := permissioncol.RoleColumns
	roleIDs := make([]pulid.ID, 0)
	if err := db.NewSelect().
		Model((*permission.Role)(nil)).
		Column(roleCols.ID.Bare()).
		Where(roleCols.IsSystem.IsTrue()).
		Where(roleCols.Name.Eq(), AdministratorRoleName).
		Scan(ctx, &roleIDs); err != nil {
		return nil, fmt.Errorf("list administrator roles: %w", err)
	}

	result := &AdminReconcileResult{RolesChecked: len(roleIDs)}
	if len(roleIDs) == 0 {
		return result, nil
	}

	existing := make([]*permission.ResourcePermission, 0)
	if err := db.NewSelect().
		Model(&existing).
		Where(permissioncol.ResourcePermissionColumns.RoleID.In(), bun.List(roleIDs)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list administrator grants: %w", err)
	}

	byRole := make(map[pulid.ID][]*permission.ResourcePermission, len(roleIDs))
	for _, rp := range existing {
		byRole[rp.RoleID] = append(byRole[rp.RoleID], rp)
	}

	now := timeutils.NowUnix()
	for _, roleID := range roleIDs {
		plan := planAdminGrants(registry, roleID, byRole[roleID], now)
		if len(plan.add) > 0 {
			if _, err := db.NewInsert().Model(&plan.add).Exec(ctx); err != nil {
				return nil, fmt.Errorf("grant new resources to role %s: %w", roleID, err)
			}
		}
		for _, rp := range plan.extend {
			if _, err := db.NewUpdate().
				Model(rp).
				Column(
					permissioncol.ResourcePermissionColumns.Operations.Bare(),
					permissioncol.ResourcePermissionColumns.UpdatedAt.Bare(),
				).
				WherePK().
				Exec(ctx); err != nil {
				return nil, fmt.Errorf("grant new operations to role %s: %w", roleID, err)
			}
		}
		result.GrantsAdded += len(plan.add)
		result.GrantsExtended += len(plan.extend)
	}

	return result, nil
}

func planAdminGrants(
	registry *permission.Registry,
	roleID pulid.ID,
	held []*permission.ResourcePermission,
	now int64,
) adminGrantPlan {
	byResource := make(map[string]*permission.ResourcePermission, len(held))
	for _, rp := range held {
		byResource[rp.Resource] = rp
	}

	definitions := registry.All()
	slices.SortFunc(definitions, func(a, b *permission.ResourceDefinition) int {
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
			plan.add = append(plan.add, &permission.ResourcePermission{
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
