package shipmenteventrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultListLimit = 25
	maxListLimit     = 200
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.ShipmentEventRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-event-repository"),
	}
}

func (r *repository) Insert(ctx context.Context, entity *shipmentevent.Event) error {
	if _, err := r.db.DB().NewInsert().Model(entity).Exec(ctx); err != nil {
		r.l.Error("failed to insert shipment event", zap.Error(err))
		return err
	}
	return nil
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListShipmentEventsRequest,
) ([]*shipmentevent.Event, error) {
	limit := req.Limit
	switch {
	case limit <= 0:
		limit = defaultListLimit
	case limit > maxListLimit:
		limit = maxListLimit
	}

	entities := make([]*shipmentevent.Event, 0, limit)

	q := r.db.DB().NewSelect().
		Model(&entities).
		Where("se.organization_id = ?", req.TenantInfo.OrgID).
		Where("se.business_unit_id = ?", req.TenantInfo.BuID).
		Order("se.occurred_at DESC").
		Limit(limit).
		Relation("Actor").
		Relation("Shipment")

	if req.ShipmentID.IsNotNil() {
		q = q.Where("se.shipment_id = ?", req.ShipmentID)
	}
	if len(req.Types) > 0 {
		q = q.Where("se.type IN (?)", bun.List(req.Types))
	}
	if req.Before > 0 {
		q = q.Where("se.occurred_at < ?", req.Before)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list shipment events", zap.Error(err))
		return nil, err
	}

	return entities, nil
}
