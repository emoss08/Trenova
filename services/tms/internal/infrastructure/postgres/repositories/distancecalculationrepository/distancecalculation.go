package distancecalculationrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distancecalculation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func New(p Params) repositories.DistanceCalculationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.distance-calculation-repository"),
	}
}

func (r *repository) CreateRun(ctx context.Context, entity *distancecalculation.Run) error {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		r.l.Error("failed to create distance calculation run", zap.Error(err))
		return err
	}
	return nil
}

func (r *repository) UpdateMoveDistance(ctx context.Context, move *shipment.ShipmentMove) error {
	result, err := r.db.DBForContext(ctx).NewUpdate().
		Model(move).
		Column(
			"distance",
			"distance_source",
			"distance_provider",
			"distance_calculated_at",
			"distance_route_signature",
			"distance_data_version",
			"distance_routing_type",
			"distance_units",
			"distance_metadata",
			"updated_at",
		).
		Where("id = ?", move.ID).
		Where("organization_id = ?", move.OrganizationID).
		Where("business_unit_id = ?", move.BusinessUnitID).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update shipment move distance", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "ShipmentMove", move.ID.String())
}
