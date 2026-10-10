package planservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func OrUnlimited(plans platformplanports.PlanService) platformplanports.PlanService {
	if plans == nil {
		return NewUnlimited()
	}

	return plans
}

func RequireCapability(
	ctx context.Context,
	plans platformplanports.PlanService,
	tenantInfo pagination.TenantInfo,
	capability platformplan.Capability,
) error {
	if plans == nil {
		return nil
	}

	return plans.RequireCapability(ctx, tenantInfo, capability)
}

func Allows(
	ctx context.Context,
	plans platformplanports.PlanService,
	tenantInfo pagination.TenantInfo,
	capability platformplan.Capability,
) (bool, error) {
	err := RequireCapability(ctx, plans, tenantInfo, capability)
	if err == nil {
		return true, nil
	}
	if errortypes.IsPlanRestrictionError(err) {
		return false, nil
	}

	return false, err
}

func RequireWritable(
	ctx context.Context,
	plans platformplanports.PlanService,
	tenantInfo pagination.TenantInfo,
) error {
	if plans == nil {
		return nil
	}

	return plans.RequireWritable(ctx, tenantInfo)
}
