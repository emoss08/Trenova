package permissionports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PricingTemplatePlan interface {
	OffPlan(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		templateIDs ...pulid.ID,
	) (map[pulid.ID]struct{}, error)
	RefuseOffPlan(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		templateIDs ...pulid.ID,
	) error
}
