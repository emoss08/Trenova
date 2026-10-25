package shipmenteventrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent/shipmenteventdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultListLimit = 25
	maxListLimit     = 200
)

var eventErrors = dberror.Mapping{Entity: "Shipment event"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) shipmenteventports.ShipmentEventRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-event-repository"),
	}
}

func (r *repository) Insert(ctx context.Context, entity *shipmentevent.Event) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := shipmenteventdb.Events.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to insert shipment event", zap.Error(err))
			return eventErrors.Map(err, entity.ID.String())
		}
		return nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req shipmenteventports.GetShipmentEventByIDRequest,
) (*shipmentevent.Event, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*shipmentevent.Event, error) {
		se := shipmenteventdb.Events
		entity, err := se.Where(se.ID.Eq(req.ID)).Load(se.Shipment()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, eventErrors.Map(err, req.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) List(
	ctx context.Context,
	req *shipmenteventports.ListShipmentEventsRequest,
) ([]*shipmentevent.Event, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipmentevent.Event, error) {
		limit := req.Limit
		switch {
		case limit <= 0:
			limit = defaultListLimit
		case limit > maxListLimit:
			limit = maxListLimit
		}

		se := shipmenteventdb.Events
		q := se.Load(se.Actor(), se.Shipment()).Order(se.OccurredAt.Desc()).Limit(limit)
		if req.ShipmentID.IsNotNil() {
			q = q.Where(se.ShipmentID.Eq(req.ShipmentID))
		}
		if len(req.Types) > 0 {
			q = q.Where(se.Type.In(req.Types...))
		}
		if req.Before > 0 {
			q = q.Where(se.OccurredAt.Lt(req.Before))
		}

		entities, err := q.List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list shipment events", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
