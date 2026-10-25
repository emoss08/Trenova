package servicetyperepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/servicetypeports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var serviceTypeErrors = dberror.Mapping{
	Entity: "ServiceType",
	Unique: []dberror.Constraint{
		{
			Name:    "idx_service_types_code",
			Field:   "code",
			Message: "Service type with this code already exists in your organization",
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

func New(p Params) servicetypeports.ServiceTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.service-type-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *servicetypeports.ListServiceTypesRequest,
) (*pagination.ListResult[*servicetype.ServiceType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*servicetype.ServiceType], error) {
			st := servicetypedb.ServiceTypes
			result, err := listquery.List(ctx, r.db.Keel(ctx), st.Query(),
				&listquery.Request[servicetype.ServiceType]{
					Filter: req.Filter,
					Then:   []keel.Order{st.CreatedAt.Desc()},
				})
			if err != nil {
				r.l.Error("failed to list service types", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *servicetypeports.ListServiceTypesConnectionRequest,
) (*pagination.CursorListResult[*servicetype.ServiceType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*servicetype.ServiceType], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				servicetypedb.ServiceTypes.Query(),
				&listquery.Request[servicetype.ServiceType]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.ServiceTypeColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list service types", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *servicetype.ServiceType,
) (*servicetype.ServiceType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*servicetype.ServiceType, error) {
		if err := servicetypedb.ServiceTypes.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to create service type",
				zap.String("code", entity.Code),
				zap.Error(err),
			)
			return nil, serviceTypeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *servicetype.ServiceType,
) (*servicetype.ServiceType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*servicetype.ServiceType, error) {
		if err := servicetypedb.ServiceTypes.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update service type",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, serviceTypeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req servicetypeports.GetServiceTypeByIDRequest,
) (*servicetype.ServiceType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*servicetype.ServiceType, error) {
		entity, err := servicetypedb.ServiceTypes.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, serviceTypeErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req servicetypeports.GetServiceTypesByIDsRequest,
) ([]*servicetype.ServiceType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*servicetype.ServiceType, error) {
		if len(req.ServiceTypeIDs) == 0 {
			return []*servicetype.ServiceType{}, nil
		}

		st := servicetypedb.ServiceTypes
		entities, err := st.Where(st.ID.In(req.ServiceTypeIDs...)).
			Limit(len(req.ServiceTypeIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get service types", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *servicetypeports.BulkUpdateServiceTypeStatusRequest,
) ([]*servicetype.ServiceType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*servicetype.ServiceType, error) {
		st := servicetypedb.ServiceTypes
		entities, err := st.Where(st.ID.In(req.ServiceTypeIDs...)).
			UpdateReturning(ctx, r.db.Keel(ctx), st.Status.To(req.Status))
		if err != nil {
			r.l.Error("failed to bulk update service type status", zap.Error(err))
			return nil, err
		}

		err = dberror.CheckBulkAffected(int64(len(entities)), "ServiceType", req.ServiceTypeIDs)
		if err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *servicetypeports.ServiceTypeSelectOptionsRequest,
) (*pagination.ListResult[*servicetype.ServiceType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*servicetype.ServiceType], error) {
			st := servicetypedb.ServiceTypes
			q := st.Pick(st.ID, st.Status, st.Code, st.Description, st.Color).
				Where(st.Status.Eq(domaintypes.StatusActive))

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(st.Code.ContainsFold(s), st.Description.ContainsFold(s))
				},
			)
		},
	)
}
