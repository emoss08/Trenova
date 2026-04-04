package servicetyperepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
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
	cols := buncolgen.ServiceTypeColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.ServiceTypeTable.Alias,
		req.Filter,
		(*servicetype.ServiceType)(nil),
	)

	return q.Apply(buncolgen.ServiceTypeApplyTenant(req.Filter.TenantInfo)).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.CreatedAt.OrderDesc())
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
	cols := buncolgen.ServiceTypeColumns

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
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
	cols := buncolgen.ServiceTypeColumns
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ServiceTypeScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
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
	cols := buncolgen.ServiceTypeColumns

	return dbhelper.SelectOptions[*servicetype.ServiceType](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.Code,
				cols.Description,
				cols.Color,
			},
			OrgColumnRef: &cols.OrganizationID,
			BuColumnRef:  &cols.BusinessUnitID,
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(cols.Status.Eq(), domaintypes.StatusActive)
			},
			EntityName:       "ServiceType",
			SearchColumnRefs: []buncolgen.Column{cols.Code, cols.Description},
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
	cols := buncolgen.ServiceTypeColumns
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ServiceTypeScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.In(), bun.In(req.ServiceTypeIDs))
		}).
		Set(cols.Status.Set(), req.Status).
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
	cols := buncolgen.ServiceTypeColumns
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ServiceTypeScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.In(), bun.In(req.ServiceTypeIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get service types", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "ServiceType")
	}

	return entities, nil
}
