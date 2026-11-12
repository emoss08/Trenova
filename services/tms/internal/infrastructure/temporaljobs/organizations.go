package temporaljobs

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// DefaultOrganizationPageSize is how many organizations a fan-out starts
// children for before it lists the next page.
const DefaultOrganizationPageSize = 200

// TenantPager pages through organizations in id order
// (tenantports.TenantSyncRepository).
type TenantPager interface {
	ListTenantsPage(ctx context.Context, after pulid.ID, limit int) ([]pagination.TenantInfo, error)
}

// OrganizationPage reads one page of organizations in id order after the
// cursor, from the database, so a fan-out never holds every organization at
// once. It reads one more than the page to know whether another follows.
func OrganizationPage(
	ctx context.Context,
	tenants TenantPager,
	after *TenantWorkItem,
	limit int,
) (*TenantPage, error) {
	limit = NormalizeLimit(limit, DefaultOrganizationPageSize)
	cursor := pulid.Nil
	if after != nil {
		cursor = after.OrganizationID
	}

	page, err := tenants.ListTenantsPage(ctx, cursor, limit+1)
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
