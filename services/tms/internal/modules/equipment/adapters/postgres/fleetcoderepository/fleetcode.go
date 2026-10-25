package fleetcoderepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode/fleetcodedb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/fleetcodeports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var fleetCodeErrors = dberror.Mapping{
	Entity: "FleetCode",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_fleet_codes_manager",
			Field:   "managerId",
			Message: "Manager does not exist or is not a member of your organization",
		},
	},
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) fleetcodeports.FleetCodeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fleet-code-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *fleetcodeports.ListFleetCodesRequest,
) (*pagination.ListResult[*fleetcode.FleetCode], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*fleetcode.FleetCode], error) {
			fc := fleetcodedb.FleetCodes
			result, err := listquery.List(ctx, r.db.Keel(ctx), fc.Query(),
				&listquery.Request[fleetcode.FleetCode]{
					Filter: req.Filter,
					Shape: func(q *keel.Query[fleetcode.FleetCode]) *keel.Query[fleetcode.FleetCode] {
						if req.IncludeManagerDetails {
							return q.Load(fc.Manager())
						}
						return q
					},
				})
			if err != nil {
				r.l.Error("failed to list fleet codes", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *fleetcodeports.ListFleetCodeConnectionRequest,
) (*pagination.CursorListResult[*fleetcode.FleetCode], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*fleetcode.FleetCode], error) {
			fc := fleetcodedb.FleetCodes
			result, err := listquery.Connection(ctx, r.db.Keel(ctx), fc.Query(),
				&listquery.Request[fleetcode.FleetCode]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.FleetCodeColumns,
					Shape: func(q *keel.Query[fleetcode.FleetCode]) *keel.Query[fleetcode.FleetCode] {
						return q.Load(fc.Manager())
					},
				})
			if err != nil {
				r.l.Error("failed to list fleet codes", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *fleetcode.FleetCode,
) (*fleetcode.FleetCode, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fleetcode.FleetCode, error) {
		if err := fleetcodedb.FleetCodes.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create fleet code",
				zap.String("code", entity.Code),
				zap.Error(err),
			)
			return nil, fleetCodeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *fleetcode.FleetCode,
) (*fleetcode.FleetCode, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fleetcode.FleetCode, error) {
		if err := fleetcodedb.FleetCodes.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update fleet code",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, fleetCodeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req fleetcodeports.GetFleetCodeByIDRequest,
) (*fleetcode.FleetCode, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fleetcode.FleetCode, error) {
		entity, err := fleetcodedb.FleetCodes.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, fleetCodeErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*fleetcode.FleetCode], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*fleetcode.FleetCode], error) {
			fc := fleetcodedb.FleetCodes
			q := fc.Pick(fc.ID, fc.Code, fc.Description, fc.Status, fc.ManagerID).
				Where(fc.Status.Eq(domaintypes.StatusActive)).
				Load(fc.Manager())

			return listquery.SelectOptions(ctx, r.db.Keel(ctx), q, req, func(s string) keel.Pred {
				return keel.Or(fc.Code.ContainsFold(s), fc.Description.ContainsFold(s))
			})
		},
	)
}
