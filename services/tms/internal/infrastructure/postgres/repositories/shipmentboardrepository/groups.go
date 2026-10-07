package shipmentboardrepository

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

var ErrUnsupportedGrouping = errors.New("shipment board grouping is not supported")

const (
	groupKeyAlias   = "group_key"
	groupLabelAlias = "group_label"
	groupStopAlias  = "grp_stop"
	groupCusAlias   = "grp_cus"
	groupOwnerAlias = "grp_own"
	dayKeyFormat    = "YYYY-MM-DD"
)

func (r *Repository) GroupSummary(
	ctx context.Context,
	req *repositories.SummarizeShipmentBoardGroupsRequest,
) ([]*repositories.ShipmentBoardGroupRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*repositories.ShipmentBoardGroupRow, error) {
			if req == nil {
				return nil, ErrScopeMissing
			}

			q, err := r.scoped(r.db.DBForContext(ctx), req.Scope)
			if err != nil {
				return nil, err
			}

			q, err = groupSummaryQuery(q, req.GroupBy, req.Timezone)
			if err != nil {
				return nil, err
			}

			rows := make([]*repositories.ShipmentBoardGroupRow, 0)
			if err = q.Scan(ctx, &rows); err != nil {
				r.l.Error(
					"failed to summarize shipment board groups",
					zap.String("groupBy", string(req.GroupBy)),
					zap.Error(err),
				)
				return nil, err
			}

			return rows, nil
		},
	)
}

func groupSummaryQuery(
	q *bun.SelectQuery,
	groupBy shipment.BoardGrouping,
	timezone string,
) (*bun.SelectQuery, error) {
	sp := buncolgen.ShipmentColumns
	q = q.
		ColumnExpr(buncolgen.Count(countAlias)).
		ColumnExpr(sp.TotalChargeAmount.Expr("COALESCE(SUM({}), 0) AS " + stageRevenueAlias))

	switch groupBy {
	case shipment.BoardGroupingShipDate:
		return dayGroups(q, shipment.ShipperStopIDSQL(buncolgen.ShipmentTable.Alias), timezone), nil
	case shipment.BoardGroupingDeliveryDate:
		return dayGroups(
			q,
			shipment.ConsigneeStopIDSQL(buncolgen.ShipmentTable.Alias),
			timezone,
		), nil
	case shipment.BoardGroupingCustomer:
		return customerGroups(q), nil
	case shipment.BoardGroupingOwner:
		return ownerGroups(q), nil
	default:
		return nil, ErrUnsupportedGrouping
	}
}

func dayGroups(q *bun.SelectQuery, stopIDSQL, timezone string) *bun.SelectQuery {
	start := buncolgen.StopColumns.ScheduledWindowStart.WithAlias(groupStopAlias)

	return q.
		Join("LEFT JOIN stops AS "+groupStopAlias).
		JoinOn(buncolgen.StopColumns.ID.WithAlias(groupStopAlias).Qualified()+" = "+stopIDSQL).
		ColumnExpr(
			"COALESCE(to_char(to_timestamp("+start.Qualified()+") AT TIME ZONE ?, ?), '') AS ?",
			timezone,
			dayKeyFormat,
			bun.Ident(groupKeyAlias),
		).
		ColumnExpr("'' AS ?", bun.Ident(groupLabelAlias)).
		GroupExpr("?", bun.Ident(groupKeyAlias)).
		OrderExpr("MIN(" + start.Qualified() + ") ASC NULLS LAST")
}

func customerGroups(q *bun.SelectQuery) *bun.SelectQuery {
	sp := buncolgen.ShipmentColumns
	cus := buncolgen.CustomerColumns
	id := cus.ID.WithAlias(groupCusAlias)
	name := cus.Name.WithAlias(groupCusAlias)

	return q.
		Join("LEFT JOIN customers AS " + groupCusAlias).
		JoinOn(id.EqColumn(sp.CustomerID)).
		JoinOn(cus.OrganizationID.WithAlias(groupCusAlias).EqColumn(sp.OrganizationID)).
		JoinOn(cus.BusinessUnitID.WithAlias(groupCusAlias).EqColumn(sp.BusinessUnitID)).
		ColumnExpr(sp.CustomerID.As(groupKeyAlias)).
		ColumnExpr(name.Expr("COALESCE({}, '') AS " + groupLabelAlias)).
		GroupExpr(sp.CustomerID.Qualified()).
		GroupExpr(name.Qualified()).
		OrderExpr(name.Qualified() + " ASC NULLS LAST").
		OrderExpr(sp.CustomerID.OrderAsc())
}

func ownerGroups(q *bun.SelectQuery) *bun.SelectQuery {
	sp := buncolgen.ShipmentColumns
	usr := buncolgen.UserColumns
	name := usr.Name.WithAlias(groupOwnerAlias)

	return q.
		Join("LEFT JOIN users AS " + groupOwnerAlias).
		JoinOn(usr.ID.WithAlias(groupOwnerAlias).EqColumn(sp.OwnerID)).
		ColumnExpr(sp.OwnerID.Expr("COALESCE({}, '') AS " + groupKeyAlias)).
		ColumnExpr(name.Expr("COALESCE({}, '') AS " + groupLabelAlias)).
		GroupExpr(sp.OwnerID.Qualified()).
		GroupExpr(name.Qualified()).
		OrderExpr(name.Qualified() + " ASC NULLS LAST").
		OrderExpr(sp.OwnerID.Qualified() + " ASC NULLS LAST")
}
