package ratematrixrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix/ratematrixdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const busyMessage = "Rate matrix is busy. Retry the request."

var rateMatrixErrors = dberror.Mapping{Entity: "RateMatrix"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ratematrixports.RateMatrixRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.ratematrix-repository"),
	}
}

func draftDimensions() keel.Loader {
	d := ratematrixdb.RateMatrixes.Dimensions()
	return d.Where(d.VersionNumber.Eq(ratematrix.DraftVersion)).Order(d.Position.Asc())
}

func (r *repository) List(
	ctx context.Context,
	req *ratematrixports.ListRateMatrixRequest,
) (*pagination.ListResult[*ratematrix.RateMatrix], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratematrix.RateMatrix], error) {
			rm := ratematrixdb.RateMatrixes
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				rm.Query(),
				&listquery.Request[ratematrix.RateMatrix]{
					Filter: req.Filter,
					Then:   []keel.Order{rm.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list rate matrices", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *ratematrixports.ListRateMatrixConnectionRequest,
) (*pagination.CursorListResult[*ratematrix.RateMatrix], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*ratematrix.RateMatrix], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				ratematrixdb.RateMatrixes.Query(),
				&listquery.Request[ratematrix.RateMatrix]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.RateMatrixColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list rate matrices", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func rateMatrixQuery(
	req *ratematrixports.GetRateMatrixByIDRequest,
) *keel.Query[*ratematrix.RateMatrix] {
	rm := ratematrixdb.RateMatrixes
	q := rm.Where(rm.ID.Eq(req.RateMatrixID))
	if req.IncludeDimensions {
		q = q.Load(draftDimensions())
	}
	if req.ForUpdate {
		q = q.Lock(keel.ForUpdate)
	}

	return q
}

func (r *repository) GetByID(
	ctx context.Context,
	req *ratematrixports.GetRateMatrixByIDRequest,
) (*ratematrix.RateMatrix, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ratematrix.RateMatrix, error) {
		entity, err := rateMatrixQuery(req).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, rateMatrixErrors.Map(err, req.RateMatrixID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req *ratematrixports.GetRateMatricesByIDsRequest,
) ([]*ratematrix.RateMatrix, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ratematrix.RateMatrix, error) {
		if len(req.RateMatrixIDs) == 0 {
			return []*ratematrix.RateMatrix{}, nil
		}

		rm := ratematrixdb.RateMatrixes
		entities, err := rm.Where(rm.ID.In(req.RateMatrixIDs...)).
			Limit(len(req.RateMatrixIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get rate matrices by ids", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
) (*ratematrix.RateMatrix, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratematrix.RateMatrix, error) {
		err := ratematrixdb.RateMatrixes.Create(ctx, r.db.Keel(ctx), entity)
		if err == nil {
			stampDimensions(entity, false)
			err = r.insertDimensions(ctx, entity)
		}
		if err != nil {
			r.l.Error(
				"failed to create rate matrix",
				zap.String("code", entity.Code),
				zap.Error(err),
			)
			return nil, mapRateMatrixError(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateHeader(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
) (*ratematrix.RateMatrix, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratematrix.RateMatrix, error) {
		if err := ratematrixdb.RateMatrixes.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update rate matrix header",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, rateMatrixErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
) (*ratematrix.RateMatrix, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratematrix.RateMatrix, error) {
		db := r.db.Keel(ctx)
		d := ratematrixdb.RateMatrixDimensions

		err := ratematrixdb.RateMatrixes.Save(ctx, db, entity)
		if err == nil {
			_, err = d.Where(
				d.RateMatrixID.Eq(entity.ID),
				d.VersionNumber.Eq(ratematrix.DraftVersion),
			).Delete(ctx, db)
		}
		if err == nil {
			stampDimensions(entity, true)
			err = r.insertDimensions(ctx, entity)
		}
		if err != nil {
			r.l.Error("failed to update rate matrix",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, mapRateMatrixError(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) insertDimensions(ctx context.Context, entity *ratematrix.RateMatrix) error {
	if len(entity.Dimensions) == 0 {
		return nil
	}

	return ratematrixdb.RateMatrixDimensions.CreateMany(ctx, r.db.Keel(ctx), entity.Dimensions)
}

func (r *repository) Delete(
	ctx context.Context,
	req *ratematrixports.GetRateMatrixByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := ratematrixdb.RateMatrixes.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&ratematrix.RateMatrix{ID: req.RateMatrixID},
		)
		if err != nil {
			r.l.Error("failed to delete rate matrix",
				zap.String("id", req.RateMatrixID.String()),
				zap.Error(err),
			)
			return rateMatrixErrors.Map(err, req.RateMatrixID.String())
		}

		return nil
	})
}

func selectOptionsQuery() *keel.Query[*ratematrix.RateMatrix] {
	rm := ratematrixdb.RateMatrixes

	return rm.Pick(
		rm.ID,
		rm.Code,
		rm.Name,
		rm.Description,
		rm.ValueType,
		rm.Status,
		rm.PublishedVersionNumber,
	).Where(rm.PublishedVersionNumber.Gt(0), rm.Status.Ne(formulatemplate.StatusInactive))
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*ratematrix.RateMatrix], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratematrix.RateMatrix], error) {
			rm := ratematrixdb.RateMatrixes

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				selectOptionsQuery(),
				req,
				func(s string) keel.Pred {
					return keel.Or(rm.Code.ContainsFold(s), rm.Name.ContainsFold(s))
				},
			)
		},
	)
}

func mapRateMatrixError(err error, id string) error {
	return dberror.MapRetryableTransactionError(rateMatrixErrors.Map(err, id), busyMessage)
}
