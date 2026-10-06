package shipmentboardrepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/shipmentrepository"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func NewBriefingRepository(r *Repository) repositories.ShipmentBriefingRepository { return r }

func (r *Repository) LeadingLateReason(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.ShipmentLateReason, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*repositories.ShipmentLateReason, error) {
		sp := buncolgen.ShipmentColumns
		sf := buncolgen.ServiceFailureColumns
		rc := buncolgen.ReasonCodeColumns

		reason := new(repositories.ShipmentLateReason)
		err := r.db.DBForContext(ctx).NewSelect().
			Model((*servicefailure.ServiceFailure)(nil)).
			ColumnExpr(rc.Label.As("label")).
			ColumnExpr(sf.ShipmentID.Expr("COUNT(DISTINCT {}) AS count")).
			Join("JOIN "+buncolgen.ShipmentTable.As(buncolgen.ShipmentTable.Alias)).
			JoinOn(sp.ID.EqColumn(sf.ShipmentID)).
			JoinOn(sp.OrganizationID.EqColumn(sf.OrganizationID)).
			JoinOn(sp.BusinessUnitID.EqColumn(sf.BusinessUnitID)).
			Join("JOIN "+buncolgen.ReasonCodeTable.As(buncolgen.ReasonCodeTable.Alias)).
			JoinOn(rc.ID.EqColumn(sf.ReasonCodeID)).
			JoinOn(rc.OrganizationID.EqColumn(sf.OrganizationID)).
			JoinOn(rc.BusinessUnitID.EqColumn(sf.BusinessUnitID)).
			Apply(buncolgen.ServiceFailureApplyTenant(tenantInfo)).
			Where(sf.Status.In(), bun.List(servicefailure.UnresolvedStatuses())).
			Where("?", shipmentrepository.StageCondition(shipment.StageLate)).
			GroupExpr(rc.Label.Qualified()).
			OrderExpr("count DESC").
			Order(rc.Label.OrderAsc()).
			Limit(1).
			Scan(ctx, reason)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil //nolint:nilnil // no reason recorded for any late shipment
			}
			r.l.Error("failed to read the leading late reason", zap.Error(err))
			return nil, err
		}

		return reason, nil
	})
}
