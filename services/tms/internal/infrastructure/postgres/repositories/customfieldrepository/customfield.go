package customfieldrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
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

func New(p Params) repositories.CustomFieldDefinitionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.custom-field-definition-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListCustomFieldDefinitionsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"cfd",
		req.Filter,
		(*customfield.CustomFieldDefinition)(nil),
	)

	if req.ResourceType != "" {
		q = q.Where("cfd.resource_type = ?", req.ResourceType)
	}

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListCustomFieldDefinitionsRequest,
) (*pagination.ListResult[*customfield.CustomFieldDefinition], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*customfield.CustomFieldDefinition, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		Order("cfd.display_order ASC", "cfd.created_at DESC").
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count custom field definitions", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*customfield.CustomFieldDefinition]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetCustomFieldDefinitionByIDRequest,
) (*customfield.CustomFieldDefinition, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(customfield.CustomFieldDefinition)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("cfd.id = ?", req.ID).
				Where("cfd.organization_id = ?", req.TenantInfo.OrgID).
				Where("cfd.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get custom field definition", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "CustomFieldDefinition")
	}

	return entity, nil
}

func (r *repository) GetActiveByResourceType(
	ctx context.Context,
	req repositories.GetActiveByResourceTypeRequest,
) ([]*customfield.CustomFieldDefinition, error) {
	log := r.l.With(
		zap.String("operation", "GetActiveByResourceType"),
		zap.String("resourceType", req.ResourceType),
	)

	entities := make([]*customfield.CustomFieldDefinition, 0)
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("cfd.organization_id = ?", req.TenantInfo.OrgID).
				Where("cfd.business_unit_id = ?", req.TenantInfo.BuID).
				Where("cfd.resource_type = ?", req.ResourceType).
				Where("cfd.is_active = ?", true)
		}).
		Order("cfd.display_order ASC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get active custom field definitions by resource type", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *customfield.CustomFieldDefinition,
) (*customfield.CustomFieldDefinition, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
		zap.String("resourceType", entity.ResourceType),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create custom field definition", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *customfield.CustomFieldDefinition,
) (*customfield.CustomFieldDefinition, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update custom field definition", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "CustomFieldDefinition", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.GetCustomFieldDefinitionByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ID.String()),
	)

	results, err := r.db.DB().
		NewDelete().
		Model((*customfield.CustomFieldDefinition)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("id = ?", req.ID).
				Where("organization_id = ?", req.TenantInfo.OrgID).
				Where("business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete custom field definition", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "CustomFieldDefinition", req.ID.String())
}

func (r *repository) CountByResourceType(
	ctx context.Context,
	req repositories.CountByResourceTypeRequest,
) (int, error) {
	log := r.l.With(
		zap.String("operation", "CountByResourceType"),
		zap.String("resourceType", req.ResourceType),
	)

	count, err := r.db.DB().
		NewSelect().
		Model((*customfield.CustomFieldDefinition)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("cfd.organization_id = ?", req.TenantInfo.OrgID).
				Where("cfd.business_unit_id = ?", req.TenantInfo.BuID).
				Where("cfd.resource_type = ?", req.ResourceType)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count custom field definitions by resource type", zap.Error(err))
		return 0, err
	}

	return count, nil
}
