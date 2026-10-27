package holdreasonrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason/holdreasondb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/holdreasonports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var holdReasonErrors = dberror.Mapping{Entity: "HoldReason"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) holdreasonports.HoldReasonRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.hold-reason-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *holdreasonports.ListHoldReasonRequest,
) (*pagination.ListResult[*holdreason.HoldReason], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*holdreason.HoldReason], error) {
			hr := holdreasondb.HoldReasons
			result, err := listquery.List(ctx, r.db.Keel(ctx), hr.Query(),
				&listquery.Request[holdreason.HoldReason]{
					Filter: req.Filter,
					Then:   []keel.Order{hr.SortOrder.Asc(), hr.CreatedAt.Desc()},
				})
			if err != nil {
				r.l.Error("failed to list hold reasons", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *holdreasonports.ListHoldReasonConnectionRequest,
) (*pagination.CursorListResult[*holdreason.HoldReason], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*holdreason.HoldReason], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				holdreasondb.HoldReasons.Query(),
				&listquery.Request[holdreason.HoldReason]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.HoldReasonColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list hold reasons", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req holdreasonports.GetHoldReasonByIDRequest,
) (*holdreason.HoldReason, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*holdreason.HoldReason, error) {
		hr := holdreasondb.HoldReasons
		entity, err := hr.Where(hr.ID.Eq(req.ID)).
			Cache(resultcache.Reference()...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, holdReasonErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *holdreason.HoldReason,
) (*holdreason.HoldReason, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*holdreason.HoldReason, error) {
		if err := holdreasondb.HoldReasons.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create hold reason",
				zap.String("code", entity.Code),
				zap.Error(err),
			)
			return nil, holdReasonErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *holdreason.HoldReason,
) (*holdreason.HoldReason, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*holdreason.HoldReason, error) {
		if err := holdreasondb.HoldReasons.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update hold reason",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, holdReasonErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *holdreasonports.HoldReasonSelectOptionsRequest,
) (*pagination.ListResult[*holdreason.HoldReason], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*holdreason.HoldReason], error) {
			hr := holdreasondb.HoldReasons
			q := hr.Pick(hr.ID, hr.Code, hr.Label, hr.Type).
				Where(hr.Active.IsTrue()).
				Cache(resultcache.Reference()...)

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(hr.Code.ContainsFold(s), hr.Label.ContainsFold(s))
				},
			)
		},
	)
}
