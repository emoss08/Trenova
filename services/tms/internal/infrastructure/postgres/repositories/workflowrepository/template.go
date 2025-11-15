package workflowrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type TemplateParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type templateRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewTemplateRepository(p TemplateParams) repositories.WorkflowTemplateRepository {
	return &templateRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.workflow-template-repository"),
	}
}

func (r *templateRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListWorkflowTemplateRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"wft",
		req.Filter,
		(*workflow.WorkflowTemplate)(nil),
	)

	if req.Category != nil {
		q = q.Where("wft.category = ?", req.Category)
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *templateRepository) List(
	ctx context.Context,
	req *repositories.ListWorkflowTemplateRequest,
) (*pagination.ListResult[*workflow.WorkflowTemplate], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*workflow.WorkflowTemplate, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan workflow templates", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*workflow.WorkflowTemplate]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *templateRepository) GetByID(
	ctx context.Context,
	req repositories.GetWorkflowTemplateByIDRequest,
) (*workflow.WorkflowTemplate, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(workflow.WorkflowTemplate)
	err = db.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("wft.id = ?", req.ID).
				Where("wft.organization_id = ?", req.OrgID).
				Where("wft.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkflowTemplate")
	}

	return entity, nil
}

func (r *templateRepository) Create(
	ctx context.Context,
	entity *workflow.WorkflowTemplate,
) (*workflow.WorkflowTemplate, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("templateID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert workflow template", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *templateRepository) Update(
	ctx context.Context,
	entity *workflow.WorkflowTemplate,
) (*workflow.WorkflowTemplate, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("templateID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	res, err := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update workflow template", zap.Error(err))
		return nil, err
	}

	return entity, dberror.HandleUpdateError(res, "WorkflowTemplate")
}

func (r *templateRepository) Delete(
	ctx context.Context,
	id, orgID, buID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("templateID", id.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewDelete().
		Model((*workflow.WorkflowTemplate)(nil)).
		Where("id = ?", id).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete workflow template", zap.Error(err))
		return err
	}

	return nil
}

// System templates

func (r *templateRepository) GetSystemTemplates(
	ctx context.Context,
) ([]*workflow.WorkflowTemplate, error) {
	log := r.l.With(
		zap.String("operation", "GetSystemTemplates"),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	templates := make([]*workflow.WorkflowTemplate, 0)
	err = db.NewSelect().
		Model(&templates).
		Where("wft.is_system_template = ?", true).
		Order("wft.usage_count DESC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get system templates", zap.Error(err))
		return nil, err
	}

	return templates, nil
}

func (r *templateRepository) GetPublicTemplates(
	ctx context.Context,
	orgID, buID pulid.ID,
) ([]*workflow.WorkflowTemplate, error) {
	log := r.l.With(
		zap.String("operation", "GetPublicTemplates"),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	templates := make([]*workflow.WorkflowTemplate, 0)
	err = db.NewSelect().
		Model(&templates).
		Where("wft.is_public = ?", true).
		WhereOr("wft.organization_id = ? AND wft.business_unit_id = ?", orgID, buID).
		Order("wft.usage_count DESC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get public templates", zap.Error(err))
		return nil, err
	}

	return templates, nil
}

// Usage tracking

func (r *templateRepository) IncrementUsage(
	ctx context.Context,
	id, orgID, buID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "IncrementUsage"),
		zap.String("templateID", id.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewUpdate().
		Model((*workflow.WorkflowTemplate)(nil)).
		Set("usage_count = usage_count + 1").
		Where("id = ?", id).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to increment usage count", zap.Error(err))
		return err
	}

	return nil
}
