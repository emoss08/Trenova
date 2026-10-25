package distancecalculationrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecalculation"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecalculation/distancecalculationdb"
	"github.com/emoss08/trenova/internal/modules/places/ports/distancecalculationports"
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

func New(p Params) distancecalculationports.DistanceCalculationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.distance-calculation-repository"),
	}
}

func (r *repository) CreateRun(ctx context.Context, entity *distancecalculation.Run) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := distancecalculationdb.Runs.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create distance calculation run", zap.Error(err))
			return err
		}
		return nil
	})
}

func (r *repository) UpdateMoveDistance(ctx context.Context, move *shipment.ShipmentMove) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		sm := shipmentdb.ShipmentMoves
		distance := sm.Distance.ToNull()
		if move.Distance != nil {
			distance = sm.Distance.To(*move.Distance)
		}
		calculatedAt := sm.DistanceCalculatedAt.ToNull()
		if move.DistanceCalculatedAt != nil {
			calculatedAt = sm.DistanceCalculatedAt.To(*move.DistanceCalculatedAt)
		}

		affected, err := sm.Where(sm.ID.Eq(move.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			distance,
			sm.DistanceSource.To(move.DistanceSource),
			sm.DistanceProvider.To(move.DistanceProvider),
			calculatedAt,
			sm.DistanceRouteSignature.To(move.DistanceRouteSignature),
			sm.DistanceDataVersion.To(move.DistanceDataVersion),
			sm.DistanceRoutingType.To(move.DistanceRoutingType),
			sm.DistanceUnits.To(move.DistanceUnits),
			sm.DistanceMetadata.To(move.DistanceMetadata),
			sm.Version.ToExpr(sm.Version),
		)
		if err != nil {
			r.l.Error("failed to update shipment move distance", zap.Error(err))
			return err
		}

		return dberror.CheckAffected(affected, "ShipmentMove", move.ID.String())
	})
}
