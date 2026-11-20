package permissionchange

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type PlanParams struct {
	fx.In

	Notifier permissionports.PermissionChangeNotifier
	Tenants  tenantports.TenantSyncRepository
	Logger   *zap.Logger
}

type PlanResult struct {
	fx.Out

	Observer managedfeaturesports.PlanObserver `group:"managed_plan_observers"`
}

type PlanObserver struct {
	notifier permissionports.PermissionChangeNotifier
	tenants  tenantports.TenantPager
	l        *zap.Logger
}

func NewPlanObserver(p PlanParams) PlanResult {
	return PlanResult{Observer: &PlanObserver{
		notifier: p.Notifier,
		tenants:  p.Tenants,
		l:        p.Logger.Named("service.permission-change.plan"),
	}}
}

func (o *PlanObserver) PlanChanged(
	ctx context.Context,
	replacement *managedfeatures.Replacement,
) error {
	if !replacement.FeaturesChanged() && !replacement.LifecycleChanged() {
		return nil
	}

	if tenant, ok := replacement.Scope.Tenant(); ok {
		o.l.Info("an organization's plan changed; its permissions are read again",
			zap.String("organizationId", tenant.OrganizationID.String()))
		o.notifier.OrganizationChanged(ctx, pagination.TenantInfo{
			OrgID: tenant.OrganizationID,
			BuID:  tenant.BusinessUnitID,
		})
		return nil
	}

	o.l.Info("the plan changed; every organization's permissions are read again")

	return tenantports.EachTenant(
		ctx,
		o.tenants,
		func(ctx context.Context, tenant pagination.TenantInfo) error {
			o.notifier.OrganizationChanged(ctx, tenant)
			return nil
		},
	)
}
