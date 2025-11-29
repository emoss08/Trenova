package documenttemplaterepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
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

func NewTemplateRepository(p TemplateParams) repositories.DocumentTemplateRepository {
	return &templateRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.documenttemplate-repository"),
	}
}

func (r *templateRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDocumentTemplateRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(q, "dtpl", req.Filter, (*documenttemplate.DocumentTemplate)(nil))

	if req.DocumentTypeID != nil {
		q = q.Where("dtpl.document_type_id = ?", req.DocumentTypeID)
	}

	if req.Status != nil {
		q = q.Where("dtpl.status = ?", *req.Status)
	}

	if req.IsDefault != nil {
		q = q.Where("dtpl.is_default = ?", *req.IsDefault)
	}

	if req.IncludeType {
		q = q.Relation("DocumentType")
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *templateRepository) List(
	ctx context.Context,
	req *repositories.ListDocumentTemplateRequest,
) (*pagination.ListResult[*documenttemplate.DocumentTemplate], error) {
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

	entities := make([]*documenttemplate.DocumentTemplate, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan document templates", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*documenttemplate.DocumentTemplate]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *templateRepository) GetByID(
	ctx context.Context,
	req repositories.GetDocumentTemplateByIDRequest,
) (*documenttemplate.DocumentTemplate, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(documenttemplate.DocumentTemplate)
	q := db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("dtpl.id = ?", req.ID).
				Where("dtpl.organization_id = ?", req.OrgID).
				Where("dtpl.business_unit_id = ?", req.BuID)
		})

	if req.IncludeType {
		q = q.Relation("DocumentType")
	}

	if err = q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Document Template")
	}

	return entity, nil
}

func (r *templateRepository) GetByCode(
	ctx context.Context,
	orgID, buID pulid.ID,
	code string,
) (*documenttemplate.DocumentTemplate, error) {
	log := r.l.With(
		zap.String("operation", "GetByCode"),
		zap.String("code", code),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(documenttemplate.DocumentTemplate)
	err = db.NewSelect().Model(entity).
		Where("dtpl.code = ?", code).
		Where("dtpl.organization_id = ?", orgID).
		Where("dtpl.business_unit_id = ?", buID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Document Template")
	}

	return entity, nil
}

func (r *templateRepository) GetDefault(
	ctx context.Context,
	req repositories.GetDefaultTemplateRequest,
) (*documenttemplate.DocumentTemplate, error) {
	log := r.l.With(
		zap.String("operation", "GetDefault"),
		zap.String("documentTypeId", req.DocumentTypeID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(documenttemplate.DocumentTemplate)
	err = db.NewSelect().Model(entity).
		Where("dtpl.document_type_id = ?", req.DocumentTypeID).
		Where("dtpl.organization_id = ?", req.OrgID).
		Where("dtpl.business_unit_id = ?", req.BuID).
		Where("dtpl.is_default = ?", true).
		Where("dtpl.status = ?", documenttemplate.TemplateStatusActive).
		Relation("DocumentType").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Default Document Template")
	}

	return entity, nil
}

func (r *templateRepository) Create(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplate,
) (*documenttemplate.DocumentTemplate, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert document template", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *templateRepository) Update(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplate,
) (*documenttemplate.DocumentTemplate, error) {
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
		Where("dtpl.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update document template", zap.Error(rErr))
		return nil, rErr
	}

	if err = dberror.CheckRowsAffected(results, "Document Template", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *templateRepository) Delete(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplate,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	results, err := db.NewDelete().Model(entity).WherePK().Exec(ctx)
	if err != nil {
		log.Error("failed to delete document template", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "Document Template", entity.ID.String())
}

func (r *templateRepository) ClearDefaultForType(
	ctx context.Context,
	orgID, buID, documentTypeID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "ClearDefaultForType"),
		zap.String("documentTypeId", documentTypeID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewUpdate().
		Model((*documenttemplate.DocumentTemplate)(nil)).
		Set("is_default = ?", false).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("document_type_id = ?", documentTypeID).
		Where("is_default = ?", true).
		Exec(ctx)
	if err != nil {
		log.Error("failed to clear default templates", zap.Error(err))
		return err
	}

	return nil
}
