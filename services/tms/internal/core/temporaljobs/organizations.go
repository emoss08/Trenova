package temporaljobs

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
)

// DefaultOrganizationPageSize is how many organizations a fan-out starts
// children for before it lists the next page.
const DefaultOrganizationPageSize = 200

// OrganizationPage is one page of organizations in id order, starting after
// the cursor. Every organization is read at once, because the list is small
// and read in one query; the paging is for the workflow fanning out over it,
// whose history grows with every child it starts.
func OrganizationPage(
	organizations []tenant.SyncOrganization,
	after *TenantWorkItem,
	limit int,
) *TenantPage {
	limit = NormalizeLimit(limit, DefaultOrganizationPageSize)
	ordered := slices.Clone(organizations)
	slices.SortFunc(ordered, func(a, b tenant.SyncOrganization) int {
		return strings.Compare(a.ID.String(), b.ID.String())
	})

	start := 0
	if after != nil {
		cursor := after.OrganizationID.String()
		start, _ = slices.BinarySearchFunc(ordered, cursor,
			func(org tenant.SyncOrganization, id string) int {
				return strings.Compare(org.ID.String(), id)
			})
		if start < len(ordered) && ordered[start].ID.String() == cursor {
			start++
		}
	}

	end := min(start+limit, len(ordered))
	items := make([]TenantWorkItem, 0, end-start)
	for _, org := range ordered[start:end] {
		items = append(items, NewTenantWorkItem(
			pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}, 1,
		))
	}

	return &TenantPage{Tenants: items, HasMore: end < len(ordered)}
}
