package servicetyperepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"github.com/uptrace/bun"
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

func NewRepository(p Params) repositories.ServiceTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.servicetype-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListServiceTypeRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"st",
		req.Filter,
		(*servicetype.ServiceType)(nil),
	)

	if req.Status != "" {
		status, err := domain.StatusFromString(req.Status)
		if err != nil {
			r.l.Error("invalid status", zap.Error(err), zap.String("status", req.Status))
			return q
		}

		q = q.Where("st.status = ?", status)
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListServiceTypeRequest,
) (*pagination.ListResult[*servicetype.ServiceType], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*servicetype.ServiceType, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan service types", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*servicetype.ServiceType]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetServiceTypeByIDOptions,
) (*servicetype.ServiceType, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(servicetype.ServiceType)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("st.id = ?", req.ID).
				Where("st.organization_id = ?", req.OrgID).
				Where("st.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Service Type")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *servicetype.ServiceType,
) (*servicetype.ServiceType, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert service type", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *servicetype.ServiceType,
) (*servicetype.ServiceType, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	results, rErr := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("st.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update service type", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Service Type", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}
