package documenttyperepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
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

func New(p Params) repositories.DocumentTypeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-type-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDocumentTypesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"dt",
		req.Filter,
		(*documenttype.DocumentType)(nil),
	)

	q = q.Order("dt.created_at DESC")

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListDocumentTypesRequest,
) (*pagination.ListResult[*documenttype.DocumentType], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*documenttype.DocumentType, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count document types", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*documenttype.DocumentType]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *documenttype.DocumentType,
) (*documenttype.DocumentType, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create document type", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *documenttype.DocumentType,
) (*documenttype.DocumentType, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update document type", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "DocumentType", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetDocumentTypeByIDRequest,
) (*documenttype.DocumentType, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(documenttype.DocumentType)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("dt.id = ?", req.ID).
				Where("dt.organization_id = ?", req.TenantInfo.OrgID).
				Where("dt.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get document type", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DocumentType")
	}

	return entity, nil
}

func (r *repository) GetByCode(
	ctx context.Context,
	req repositories.GetDocumentTypeByCodeRequest,
) (*documenttype.DocumentType, error) {
	log := r.l.With(
		zap.String("operation", "GetByCode"),
		zap.String("code", req.Code),
	)

	entity := new(documenttype.DocumentType)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("dt.code = ?", req.Code).
				Where("dt.organization_id = ?", req.TenantInfo.OrgID).
				Where("dt.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get document type by code", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DocumentType")
	}

	return entity, nil
}

func (r *repository) GetByName(
	ctx context.Context,
	req repositories.GetDocumentTypeByNameRequest,
) (*documenttype.DocumentType, error) {
	log := r.l.With(
		zap.String("operation", "GetByName"),
		zap.String("name", req.Name),
	)

	entity := new(documenttype.DocumentType)
	normalizedName := strings.TrimSpace(req.Name)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("lower(btrim(dt.name)) = lower(btrim(?))", normalizedName).
				Where("dt.organization_id = ?", req.TenantInfo.OrgID).
				Where("dt.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get document type by name", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DocumentType")
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*documenttype.DocumentType], error) {
	return dbhelper.SelectOptions[*documenttype.DocumentType](
		ctx,
		r.db.DB(),
		req,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"code",
				"name",
				"description",
				"color",
				"document_classification",
				"document_category",
				"is_system",
			},
			OrgColumn:     "dt.organization_id",
			BuColumn:      "dt.business_unit_id",
			EntityName:    "DocumentType",
			SearchColumns: []string{"dt.code", "dt.name"},
		},
	)
}
