package temporaljobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// DefaultOrganizationPageSize is how many organizations a fan-out starts
// children for before it lists the next page.
const DefaultOrganizationPageSize = 200

// (tenantports.TenantSyncRepository).
type TenantPager interface {
	ListTenantsPage(
		ctx context.Context,
		need entitlement.Need,
		after pulid.ID,
		limit int,
	) ([]pagination.TenantInfo, error)
}

func OrganizationPage(
	ctx context.Context,
	tenants TenantPager,
	need entitlement.Need,
	after *TenantWorkItem,
	limit int,
) (*TenantPage, error) {
	if err := need.Validate(); err != nil {
		return nil, err
	}
	limit = NormalizeLimit(limit, DefaultOrganizationPageSize)
	cursor := pulid.Nil
	if after != nil {
		cursor = after.OrganizationID
	}

	page, err := tenants.ListTenantsPage(ctx, need, cursor, limit+1)
	if err != nil {
		return nil, err
	}

	hasMore := len(page) > limit
	page = page[:min(len(page), limit)]
	items := make([]TenantWorkItem, 0, len(page))
	for _, tenant := range page {
		items = append(items, NewTenantWorkItem(tenant, 1))
	}

	return &TenantPage{Tenants: items, HasMore: hasMore}, nil
}
