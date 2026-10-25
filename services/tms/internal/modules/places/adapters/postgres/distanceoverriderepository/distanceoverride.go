package distanceoverriderepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceoverride"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceoverride/distanceoverridedb"
	"github.com/emoss08/trenova/internal/modules/places/ports/distanceoverrideports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const busyMessage = "Distance override is busy. Retry the request."

var distanceOverrideErrors = dberror.Mapping{Entity: "DistanceOverride"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) distanceoverrideports.DistanceOverrideRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.distance-override-repository"),
	}
}

func listShape(
	q *keel.Query[*distanceoverride.DistanceOverride],
) *keel.Query[*distanceoverride.DistanceOverride] {
	do := distanceoverridedb.DistanceOverrides
	stops := do.IntermediateStops()
	return q.Load(
		do.Customer(),
		do.OriginLocation().State(),
		do.DestinationLocation().State(),
		stops.Order(stops.StopOrder.Asc()),
	)
}

func (r *repository) List(
	ctx context.Context,
	req *distanceoverrideports.ListDistanceOverrideRequest,
) (*pagination.ListResult[*distanceoverride.DistanceOverride], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*distanceoverride.DistanceOverride], error) {
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				distanceoverridedb.DistanceOverrides.Query(),
				&listquery.Request[distanceoverride.DistanceOverride]{
					Filter: req.Filter,
					Shape:  listShape,
				},
			)
			if err != nil {
				r.l.Error("failed to list distance overrides", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *distanceoverrideports.ListDistanceOverrideConnectionRequest,
) (*pagination.CursorListResult[*distanceoverride.DistanceOverride], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*distanceoverride.DistanceOverride], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				distanceoverridedb.DistanceOverrides.Query(),
				&listquery.Request[distanceoverride.DistanceOverride]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  listShape,
				},
			)
			if err != nil {
				r.l.Error("failed to list distance overrides", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req distanceoverrideports.GetDistanceOverrideByIDRequest,
) (*distanceoverride.DistanceOverride, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*distanceoverride.DistanceOverride, error) {
			do := distanceoverridedb.DistanceOverrides
			stops := do.IntermediateStops()
			entity, err := do.Where(do.ID.Eq(req.ID)).
				Load(
					do.OriginLocation(),
					do.DestinationLocation(),
					do.Customer(),
					stops.Order(stops.StopOrder.Asc()).Load(stops.Location()),
				).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get distance override",
					zap.String("id", req.ID.String()),
					zap.Error(err),
				)
				return nil, distanceOverrideErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *distanceoverride.DistanceOverride,
) (*distanceoverride.DistanceOverride, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*distanceoverride.DistanceOverride, error) {
			db := r.db.Keel(ctx)
			if err := distanceoverridedb.DistanceOverrides.Create(ctx, db, entity); err != nil {
				r.l.Error("failed to create distance override", zap.Error(err))
				return nil, r.mapWriteError(err, entity)
			}

			if err := r.insertStops(ctx, entity); err != nil {
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *distanceoverride.DistanceOverride,
) (*distanceoverride.DistanceOverride, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*distanceoverride.DistanceOverride, error) {
			db := r.db.Keel(ctx)
			if err := distanceoverridedb.DistanceOverrides.Save(ctx, db, entity); err != nil {
				r.l.Error(
					"failed to update distance override",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, r.mapWriteError(err, entity)
			}

			st := distanceoverridedb.DistanceOverrideStops
			if _, err := st.Where(st.DistanceOverrideID.Eq(entity.ID)).Delete(ctx, db); err != nil {
				r.l.Error("failed to replace distance override stops", zap.Error(err))
				return nil, r.mapWriteError(err, entity)
			}

			if err := r.insertStops(ctx, entity); err != nil {
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) insertStops(
	ctx context.Context,
	entity *distanceoverride.DistanceOverride,
) error {
	if len(entity.IntermediateStops) == 0 {
		return nil
	}

	entity.NormalizeIntermediateStops()
	if err := distanceoverridedb.DistanceOverrideStops.CreateMany(
		ctx,
		r.db.Keel(ctx),
		entity.IntermediateStops,
	); err != nil {
		r.l.Error("failed to insert distance override stops", zap.Error(err))
		return r.mapWriteError(err, entity)
	}

	return nil
}

func (r *repository) mapWriteError(err error, entity *distanceoverride.DistanceOverride) error {
	return dberror.MapRetryableTransactionError(
		distanceOverrideErrors.Map(err, entity.ID.String()),
		busyMessage,
	)
}

func (r *repository) Delete(
	ctx context.Context,
	req distanceoverrideports.DeleteDistanceOverrideRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		do := distanceoverridedb.DistanceOverrides
		deleted, err := do.Where(do.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to delete distance override",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return distanceOverrideErrors.Map(err, req.ID.String())
		}

		return dberror.CheckFoundCount(deleted, distanceOverrideErrors.Entity)
	})
}

func (r *repository) GetByRouteSignature(
	ctx context.Context,
	_ pagination.TenantInfo,
	routeSignature string,
) (*distanceoverride.DistanceOverride, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*distanceoverride.DistanceOverride, error) {
			do := distanceoverridedb.DistanceOverrides
			entity, err := do.Where(do.RouteSignature.Eq(routeSignature)).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, distanceOverrideErrors.Map(err, routeSignature)
			}

			return entity, nil
		},
	)
}
