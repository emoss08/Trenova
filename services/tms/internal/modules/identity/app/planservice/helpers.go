package planservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func OrUnlimited(plans platformplanports.PlanService) platformplanports.PlanService {
	if plans == nil {
		return NewUnlimited()
	}

	return plans
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
