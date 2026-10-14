package factoringrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/pkg/buncolgen/invoicecol"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const sweepTenantsReason = "find organizations with factored invoices or open notices of assignment"

type tenantRow struct {
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func (r *invoiceStateRepository) ListSweepTenants(
	ctx context.Context,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, sweepTenantsReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		dba := r.db.DBForContext(ctx)
		inv := invoicecol.InvoiceColumns
		noa := factoringcol.NoaColumns

		factored := dba.NewSelect().
			Model((*invoice.Invoice)(nil)).
			ColumnExpr(inv.OrganizationID.As("organization_id")).
			ColumnExpr(inv.BusinessUnitID.As("business_unit_id")).
			Where(inv.Status.Eq(), invoice.StatusPosted).
			Where(inv.FactoringCompanyID.IsNotNull()).
			Where(inv.TotalAmountMinor.Qualified() + " > " + inv.AppliedAmountMinor.Qualified())
		noticed := dba.NewSelect().
			Model((*factoring.Noa)(nil)).
			ColumnExpr(noa.OrganizationID.As("organization_id")).
			ColumnExpr(noa.BusinessUnitID.As("business_unit_id")).
			Where(noa.Status.Ne(), factoring.NoaStatusReleased)

		rows := make([]tenantRow, 0, 8)
		if err := dba.NewSelect().
			TableExpr("(?) AS tenants", factored.Union(noticed)).
			ColumnExpr("tenants.organization_id, tenants.business_unit_id").
			Scan(ctx, &rows); err != nil {
			return nil, fmt.Errorf("list factoring sweep tenants: %w", err)
		}

		tenants := make([]pagination.TenantInfo, 0, len(rows))
		for _, row := range rows {
			tenants = append(tenants, pagination.TenantInfo{
				OrgID: row.OrganizationID,
				BuID:  row.BusinessUnitID,
			})
		}
		return tenants, nil
	})
}
