package factoringrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const sweepTenantsReason = "find organizations with factored invoices or open notices of assignment"

type tenantRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func (r *invoiceStateRepository) ListSweepTenants(
	ctx context.Context,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, sweepTenantsReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		rows, err := keel.Into[tenantRow](ctx, r.db.Keel(ctx), sweepTenants())
		if err != nil {
			return nil, err
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

func sweepTenants() keel.Compound {
	inv := invoicedb.Invoices
	noa := factoringdb.Noas
	return keel.Union(
		inv.AllTenants().
			Pick(inv.OrganizationID, inv.BusinessUnitID).
			Where(
				inv.Status.Eq(invoice.StatusPosted),
				inv.FactoringCompanyID.IsNotNull(),
				keel.DynOf(inv.TotalAmountMinor).Gt(inv.AppliedAmountMinor),
			),
		noa.AllTenants().
			Pick(noa.OrganizationID, noa.BusinessUnitID).
			Where(noa.Status.Ne(factoring.NoaStatusReleased)),
	)
}
