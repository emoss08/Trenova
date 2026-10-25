package customertrackingrepository

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type tenantRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func tenantsOf(rows []tenantRow) []pagination.TenantInfo {
	tenants := make([]pagination.TenantInfo, 0, len(rows))
	for _, row := range rows {
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: row.OrganizationID,
			BuID:  row.BusinessUnitID,
		})
	}
	return tenants
}
