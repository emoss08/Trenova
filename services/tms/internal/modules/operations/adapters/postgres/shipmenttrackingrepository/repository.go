package shipmenttrackingrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
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

		sp := shipmentdb.Shipments
		moves := sp.Moves()
		stops := moves.Stops()
		location := stops.Location()

		entities, err := sp.Where(sp.ID.In(ids...)).
			Load(
				sp.Customer(),
				moves.Order(moves.Sequence.Asc()).Load(
					stops.Order(stops.Sequence.Asc()).Load(location.Load(location.State())),
				),
			).
			Limit(len(ids)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to load tracking shipments", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
