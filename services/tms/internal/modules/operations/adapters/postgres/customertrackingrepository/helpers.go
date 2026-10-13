package customertrackingrepository

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

const (
	orgColumn = "organization_id"
	buColumn  = "business_unit_id"
)

type tenantRow struct {
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
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

func pgArray(values []string) any {
	if values == nil {
		values = []string{}
	}
	return pgdialect.Array(values)
}

func setNullable(q *bun.UpdateQuery, column, value string) *bun.UpdateQuery {
	if value == "" {
		return q.Set("? = NULL", bun.Ident(column))
	}
	return q.Set("? = ?", bun.Ident(column), value)
}

func setNullableID(q *bun.UpdateQuery, column string, value pulid.ID) *bun.UpdateQuery {
	if value.IsNil() {
		return q.Set("? = NULL", bun.Ident(column))
	}
	return q.Set("? = ?", bun.Ident(column), value)
}
