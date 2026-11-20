package permission

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

type planGate struct {
	*onplan.Memo
}

func (e *engine) newPlanGate() planGate {
	return planGate{Memo: e.plan.Memo()}
}

func planScope(ctx context.Context, orgID, buID pulid.ID) context.Context {
	if orgID.IsNil() {
		return ctx
	}
	if tenant, ok := dbscope.TenantFrom(ctx); ok && tenant.OrganizationID == orgID {
		return ctx
	}
	return dbscope.WithTenant(ctx, dbscope.Tenant{OrganizationID: orgID, BusinessUnitID: buID})
}

func (p planGate) withhold(ctx context.Context, permissions map[string]uint32) error {
	for resource := range permissions {
		_, off, err := p.Resource(ctx, resource)
		if err != nil {
			return err
		}
		if off {
			delete(permissions, resource)
		}
	}

	return nil
}

func (p planGate) markEffective(
	ctx context.Context,
	effective *permissionports.EffectivePermissions,
) error {
	for resource, granted := range effective.Resources {
		denial, off, err := p.Resource(ctx, resource)
		if err != nil {
			return err
		}
		if off {
			granted.PlanDenial = &denial
			effective.Resources[resource] = granted
		}
	}

	return nil
}

func (p planGate) offPlan(
	ctx context.Context,
	resources ...permission.Resource,
) (map[permission.Resource]bool, error) {
	off := make(map[permission.Resource]bool, len(resources))
	for _, resource := range resources {
		if _, asked := off[resource]; asked {
			continue
		}
		_, denied, err := p.Resource(ctx, resource.String())
		if err != nil {
			return nil, err
		}
		off[resource] = denied
	}

	return off, nil
}

func deniedByPlan(
	denial permissionports.PlanDenial,
	start time.Time,
) *permissionports.PermissionCheckResult {
	return &permissionports.PermissionCheckResult{
		Allowed:       false,
		Reason:        permissionports.ReasonNotOnPlan,
		PlanDenial:    &denial,
		CheckDuration: time.Since(start).Milliseconds(),
	}
}

func offPlanDetail(
	perms *permissionports.CachedPermissions,
	resource string,
	denial permissionports.PlanDenial,
) *permissionports.ResourcePermissionDetail {
	return &permissionports.ResourcePermissionDetail{
		Resource:       resource,
		Operations:     []permission.Operation{},
		MaxSensitivity: permission.FieldSensitivity(perms.MaxSensitivity),
		PlanDenial:     &denial,
	}
}
