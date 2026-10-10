package shipmenttrackingrepository

import (
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const MaxTrackingShipments = 500

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) shipmentports.ShipmentTrackingRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.shipment-tracking-repository")}
}

func (r *repository) ListTrackingShipments(
	ctx context.Context,
	req *shipmentports.ListTrackingShipmentsRequest,
) ([]*shipment.Shipment, error) {
	if len(req.ShipmentIDs) == 0 {
		return []*shipment.Shipment{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipment.Shipment, error) {
		ids := req.ShipmentIDs
		if len(ids) > MaxTrackingShipments {
			ids = ids[:MaxTrackingShipments]
		}

		sm := shipmentcol.ShipmentMoveColumns
		stp := shipmentcol.StopColumns
		rel := shipmentcol.ShipmentRelations
		moves := shipmentcol.ShipmentMoveRelations
		stops := shipmentcol.StopRelations

		entities := make([]*shipment.Shipment, 0, len(ids))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return shipmentcol.ShipmentScopeTenant(sq, req.TenantInfo).
					Where(shipmentcol.ShipmentColumns.ID.In(), bun.List(ids))
			}).
			Relation(rel.Customer).
			RelationWithOpts(rel.Moves, bun.RelationOpts{
				Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Order(sm.Sequence.OrderAsc())
				},
			}).
			RelationWithOpts(buncolgen.Rel(rel.Moves, moves.Stops), bun.RelationOpts{
				Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Order(stp.Sequence.OrderAsc())
				},
			}).
			Relation(buncolgen.Rel(rel.Moves, moves.Stops, stops.Location)).
			Relation(buncolgen.Rel(
				rel.Moves,
				moves.Stops,
				stops.Location,
				locationcol.LocationRelations.State,
			)).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to load tracking shipments", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
