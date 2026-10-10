package shipmentboardrepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/pkg/buncolgen/servicefailurecol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func NewBriefingRepository(r *Repository) shipmentports.ShipmentBriefingRepository { return r }

func (r *Repository) LeadingLateReason(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*shipmentports.ShipmentLateReason, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentports.ShipmentLateReason, error) {
			sp := shipmentcol.ShipmentColumns
			sf := servicefailurecol.ServiceFailureColumns
			rc := servicefailurecol.ReasonCodeColumns

			reason := new(shipmentports.ShipmentLateReason)
			err := r.db.DBForContext(ctx).NewSelect().
				Model((*servicefailure.ServiceFailure)(nil)).
				ColumnExpr(rc.Label.As("label")).
				ColumnExpr(sf.ShipmentID.Expr("COUNT(DISTINCT {}) AS count")).
				Join("JOIN "+shipmentcol.ShipmentTable.As(shipmentcol.ShipmentTable.Alias)).
				JoinOn(sp.ID.EqColumn(sf.ShipmentID)).
				JoinOn(sp.OrganizationID.EqColumn(sf.OrganizationID)).
				JoinOn(sp.BusinessUnitID.EqColumn(sf.BusinessUnitID)).
				Join("JOIN "+servicefailurecol.ReasonCodeTable.As(servicefailurecol.ReasonCodeTable.Alias)).
				JoinOn(rc.ID.EqColumn(sf.ReasonCodeID)).
				JoinOn(rc.OrganizationID.EqColumn(sf.OrganizationID)).
				JoinOn(rc.BusinessUnitID.EqColumn(sf.BusinessUnitID)).
				Apply(servicefailurecol.ServiceFailureApplyTenant(tenantInfo)).
				Where(sf.Status.In(), bun.List(servicefailure.UnresolvedStatuses())).
				Where("?", shipmentrepository.StageCondition(shipment.StageLate)).
				GroupExpr(rc.Label.Qualified()).
				OrderExpr("count DESC").
				Order(rc.Label.OrderAsc()).
				Limit(1).
				Scan(ctx, reason)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, nil
				}
				r.l.Error("failed to read the leading late reason", zap.Error(err))
				return nil, err
			}

			return reason, nil
		},
	)
}
