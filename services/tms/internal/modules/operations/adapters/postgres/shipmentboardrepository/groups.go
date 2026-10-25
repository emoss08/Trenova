package shipmentboardrepository

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"go.uber.org/zap"
)

var ErrUnsupportedGrouping = errors.New("shipment board grouping is not supported")

const (
	groupStopAlias = "grp_stop"
	endStopAlias   = "end_stp"
	endMoveAlias   = "end_sm"
	dayKeyFormat   = "YYYY-MM-DD"
)

func (r *Repository) GroupSummary(
	ctx context.Context,
	req *shipmentports.SummarizeShipmentBoardGroupsQuery,
) ([]*shipmentports.ShipmentBoardGroupRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.ShipmentBoardGroupRow, error) {
			if req == nil {
				return nil, ErrScopeMissing
			}

			q, err := scoped(req.Scope)
			if err != nil {
				return nil, err
			}

			q, err = groupSummaryQuery(q, req.GroupBy, req.Timezone)
			if err != nil {
				return nil, err
			}

			rows, err := keel.Into[shipmentports.ShipmentBoardGroupRow](ctx, r.db.Keel(ctx), q)
			if err != nil {
				r.l.Error(
					"failed to summarize shipment board groups",
					zap.String("groupBy", string(req.GroupBy)),
					zap.Error(err),
				)
				return nil, err
			}

			return pointers(rows), nil
		},
	)
}

func groupSummaryQuery(
	q *keel.Query[shipment.Shipment],
	groupBy shipment.BoardGrouping,
	timezone string,
) (*keel.Query[shipment.Shipment], error) {
	switch groupBy {
	case shipment.BoardGroupingShipDate:
		return dayGroups(q, shipment.PickupStopTypes(), false, timezone), nil
	case shipment.BoardGroupingDeliveryDate:
		return dayGroups(q, shipment.DeliveryStopTypes(), true, timezone), nil
	case shipment.BoardGroupingCustomer:
		return customerGroups(q), nil
	case shipment.BoardGroupingOwner:
		return ownerGroups(q), nil
	default:
		return nil, ErrUnsupportedGrouping
	}
}

func endStopID(types []shipment.StopType, last bool) keel.Expr {
	sp := shipmentdb.Shipments
	st := shipmentdb.Stops.As(endStopAlias)
	sm := shipmentdb.ShipmentMoves.As(endMoveAlias)

	order := []keel.Order{
		sm.Sequence.Asc(),
		st.Sequence.Asc(),
		keel.Asc(keel.Raw(`? COLLATE "C"`, st.ID)),
	}
	if last {
		order = []keel.Order{
			sm.Sequence.Desc(),
			st.Sequence.Desc(),
			keel.Desc(keel.Raw(`? COLLATE "C"`, st.ID)),
		}
	}

	return keel.Scalar(sm.Pick(st.ID).
		Join(st, st.ShipmentMoveID.EqCol(sm.ID)).
		Where(sm.ShipmentID.EqCol(sp.ID), st.Type.In(types...)).
		Order(order...).
		Limit(1))
}

func dayGroups(
	q *keel.Query[shipment.Shipment],
	types []shipment.StopType,
	last bool,
	timezone string,
) *keel.Query[shipment.Shipment] {
	gs := shipmentdb.Stops.As(groupStopAlias)
	key := keel.Raw(
		"COALESCE(to_char(to_timestamp(?) AT TIME ZONE ?, ?), '')",
		gs.ScheduledWindowStart,
		timezone,
		dayKeyFormat,
	)

	return q.LeftJoin(gs, gs.ID.EqCol(endStopID(types, last))).
		Pick(key, keel.Raw("''"), keel.Count(), revenueSum()).
		Group(key).
		Order(keel.Asc(keel.Raw("MIN(?)", gs.ScheduledWindowStart)).NullsLast())
}

func customerGroups(q *keel.Query[shipment.Shipment]) *keel.Query[shipment.Shipment] {
	sp := shipmentdb.Shipments
	cus := sp.Customer()

	return q.Pick(sp.CustomerID, keel.Raw("COALESCE(?, '')", cus.Name), keel.Count(), revenueSum()).
		Group(sp.CustomerID, cus.Name).
		Order(cus.Name.Asc().NullsLast(), sp.CustomerID.Asc())
}

func ownerGroups(q *keel.Query[shipment.Shipment]) *keel.Query[shipment.Shipment] {
	sp := shipmentdb.Shipments
	owner := sp.Owner()

	return q.Pick(
		keel.Raw("COALESCE(?, '')", sp.OwnerID),
		keel.Raw("COALESCE(?, '')", owner.Name),
		keel.Count(),
		revenueSum(),
	).
		Group(sp.OwnerID, owner.Name).
		Order(owner.Name.Asc().NullsLast(), sp.OwnerID.Asc().NullsLast())
}
