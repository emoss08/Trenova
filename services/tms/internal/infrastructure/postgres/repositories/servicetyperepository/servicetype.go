package servicetyperepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

func New(p Params) repositories.ServiceTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.servicetype-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListServiceTypesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"st",
		req.Filter,
		(*servicetype.ServiceType)(nil),
	)

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListServiceTypesRequest,
) (*pagination.ListResult[*servicetype.ServiceType], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*servicetype.ServiceType, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count service types", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*servicetype.ServiceType]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *servicetype.ServiceType,
) (*servicetype.ServiceType, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Description),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create service type", zap.Error(err))
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
		zap.String("description", entity.Description),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update service type", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "ServiceType", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetServiceTypeByIDRequest,
) (*servicetype.ServiceType, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(servicetype.ServiceType)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("st.id = ?", req.ID).
				Where("st.organization_id = ?", req.TenantInfo.OrgID).
				Where("st.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get service type", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "ServiceType")
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.ServiceTypeSelectOptionsRequest,
) (*pagination.ListResult[*servicetype.ServiceType], error) {
	return dbhelper.SelectOptions[*servicetype.ServiceType](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"code",
				"description",
				"color",
			},
			OrgColumn: "st.organization_id",
			BuColumn:  "st.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("st.status = ?", domaintypes.StatusActive)
			},
			EntityName: "ServiceType",
			SearchColumns: []string{
				"st.code",
				"st.description",
			},
		},
	)
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateServiceTypeStatusRequest,
) ([]*servicetype.ServiceType, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*servicetype.ServiceType, 0, len(req.ServiceTypeIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("st.organization_id = ?", req.TenantInfo.OrgID).
				Where("st.business_unit_id = ?", req.TenantInfo.BuID).
				Where("st.id IN (?)", bun.In(req.ServiceTypeIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update service type status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "ServiceType", req.ServiceTypeIDs); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetServiceTypesByIDsRequest,
) ([]*servicetype.ServiceType, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*servicetype.ServiceType, 0, len(req.ServiceTypeIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("st.organization_id = ?", req.TenantInfo.OrgID).
				Where("st.business_unit_id = ?", req.TenantInfo.BuID).
				Where("st.id IN (?)", bun.In(req.ServiceTypeIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get service types", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "ServiceType")
	}

	return entities, nil
}
