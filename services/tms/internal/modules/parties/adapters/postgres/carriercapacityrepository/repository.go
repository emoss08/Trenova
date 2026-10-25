package carriercapacityrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carriercapacity/carriercapacitydb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carriercapacityports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	entityName      = "Carrier capacity posting"
	defaultOpenList = 500
	maxOpenList     = 2000
)

var postingErrors = dberror.Mapping{Entity: entityName}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) carriercapacityports.CarrierCapacityPostingRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.carrier-capacity-posting-repository")}
}

func postingLoads() []keel.Loader {
	p := carriercapacitydb.Postings

	return []keel.Loader{
		p.Carrier(),
		p.OriginLocation(),
		p.OriginState(),
		p.DestinationState(),
		p.EquipmentType(),
	}
}

func withPostingLoads(
	q *keel.Query[carriercapacity.Posting],
) *keel.Query[carriercapacity.Posting] {
	return q.Load(postingLoads()...)
}

func listOptions(req *carriercapacityports.ListCarrierCapacityPostingsRequest) []keel.Pred {
	p := carriercapacitydb.Postings

	return []keel.Pred{
		keel.When(req.CarrierID.IsNotNil(), p.CarrierID.Eq(req.CarrierID)),
		keel.When(req.OpenAt > 0, p.AvailableTo.Gte(req.OpenAt)),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *carriercapacityports.ListCarrierCapacityPostingsRequest,
) (*pagination.ListResult[*carriercapacity.Posting], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*carriercapacity.Posting], error) {
			p := carriercapacitydb.Postings
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				p.Where(listOptions(req)...),
				&listquery.Request[carriercapacity.Posting]{
					Filter: req.Filter,
					Shape:  withPostingLoads,
					Then:   []keel.Order{p.AvailableFrom.Asc(), p.ID.Asc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list carrier capacity postings", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *carriercapacityports.ListCarrierCapacityPostingsRequest,
) (*pagination.CursorListResult[*carriercapacity.Posting], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*carriercapacity.Posting], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				carriercapacitydb.Postings.Where(listOptions(req)...),
				&listquery.Request[carriercapacity.Posting]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withPostingLoads,
				},
			)
			if err != nil {
				r.l.Error("failed to list carrier capacity postings", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *carriercapacityports.GetCarrierCapacityPostingRequest,
) (*carriercapacity.Posting, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*carriercapacity.Posting, error) {
		p := carriercapacitydb.Postings
		entity, err := p.Where(p.ID.Eq(req.ID)).Load(postingLoads()...).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, postingErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *carriercapacity.Posting,
) (*carriercapacity.Posting, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*carriercapacity.Posting, error) {
		if err := carriercapacitydb.Postings.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create carrier capacity posting", zap.Error(err))
			return nil, postingErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *carriercapacity.Posting,
) (*carriercapacity.Posting, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*carriercapacity.Posting, error) {
		if err := carriercapacitydb.Postings.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update carrier capacity posting", zap.Error(err))
			return nil, postingErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req *carriercapacityports.DeleteCarrierCapacityPostingRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := carriercapacitydb.Postings
		affected, err := p.Where(p.ID.Eq(req.ID), p.Version.Eq(req.Version)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete carrier capacity posting", zap.Error(err))
			return err
		}

		return dberror.CheckAffected(affected, entityName, req.ID.String())
	})
}

func (r *repository) ListOpen(
	ctx context.Context,
	req *carriercapacityports.ListOpenCarrierCapacityRequest,
) ([]*carriercapacity.Posting, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*carriercapacity.Posting, error) {
		p := carriercapacitydb.Postings
		limit := req.Limit
		if limit <= 0 {
			limit = defaultOpenList
		}
		limit = min(limit, maxOpenList)

		through := max(req.Through, req.OpenAt)

		entities, err := p.Where(
			p.AvailableTo.Gte(req.OpenAt),
			p.AvailableFrom.Lte(through),
			keel.When(len(req.CarrierIDs) > 0, p.CarrierID.In(req.CarrierIDs...)),
		).
			Load(postingLoads()...).
			Order(p.AvailableFrom.Asc(), p.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list open carrier capacity", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
