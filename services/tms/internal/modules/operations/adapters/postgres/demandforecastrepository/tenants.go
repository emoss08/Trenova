package demandforecastrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol/dispatchcontroldb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultTenantLimit = 100
	maxTenantLimit     = 500
)

type tenantRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func (r *repository) ListTenants(
	ctx context.Context,
	req *demandforecastports.ListTenantsRequest,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list the organizations whose freight forecast is fitted nightly")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		dc := dispatchcontroldb.DispatchControls
		limit := min(max(req.Limit, 1), maxTenantLimit)
		if req.Limit <= 0 {
			limit = defaultTenantLimit
		}
		q := dc.AllTenants().
			Pick(dc.OrganizationID, dc.BusinessUnitID).
			Order(dc.OrganizationID.Asc(), dc.BusinessUnitID.Asc()).
			Limit(limit)
		if !req.After.OrgID.IsNil() {
			q = q.Where(
				keel.Tuple(dc.OrganizationID, dc.BusinessUnitID).
					Gt(keel.Tuple(req.After.OrgID.String(), req.After.BuID.String())),
			)
		}

		rows, err := keel.Into[tenantRow](ctx, r.db.Keel(ctx), q)
		if err != nil {
			return nil, fmt.Errorf("list forecast tenants: %w", err)
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
