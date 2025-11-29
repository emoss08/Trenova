package generateddocumentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
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

func NewRepository(p Params) repositories.GeneratedDocumentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.generateddocument-repository"),
	}
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	req *repositories.ListGeneratedDocumentRequest,
) *bun.SelectQuery {
	if req.ReferenceType != nil {
		q = q.Where("gdoc.reference_type = ?", *req.ReferenceType)
	}

	if req.ReferenceID != nil {
		q = q.Where("gdoc.reference_id = ?", req.ReferenceID)
	}

	if req.Status != nil {
		q = q.Where("gdoc.status = ?", *req.Status)
	}

	if req.IncludeType {
		q = q.Relation("DocumentType")
	}

	return q
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListGeneratedDocumentRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(q, "gdoc", req.Filter, (*documenttemplate.GeneratedDocument)(nil))

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req)
	})

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListGeneratedDocumentRequest,
) (*pagination.ListResult[*documenttemplate.GeneratedDocument], error) {
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

	entities := make([]*documenttemplate.GeneratedDocument, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan generated documents", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*documenttemplate.GeneratedDocument]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetGeneratedDocumentByIDRequest,
) (*documenttemplate.GeneratedDocument, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(documenttemplate.GeneratedDocument)
	q := db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("gdoc.id = ?", req.ID).
				Where("gdoc.organization_id = ?", req.OrgID).
				Where("gdoc.business_unit_id = ?", req.BuID)
		})

	if req.IncludeType {
		q = q.Relation("DocumentType")
	}

	if err = q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Generated Document")
	}

	return entity, nil
}

func (r *repository) GetByReference(
	ctx context.Context,
	req *repositories.GetByReferenceRequest,
) ([]*documenttemplate.GeneratedDocument, error) {
	log := r.l.With(
		zap.String("operation", "GetByReference"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*documenttemplate.GeneratedDocument, 0)
	err = db.NewSelect().Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("gdoc.organization_id = ?", req.OrgID).
				Where("gdoc.business_unit_id = ?", req.BuID).
				Where("gdoc.reference_type = ?", req.RefType).
				Where("gdoc.reference_id = ?", req.RefID)
		}).
		Order("gdoc.created_at DESC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get documents by reference", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *documenttemplate.GeneratedDocument,
) (*documenttemplate.GeneratedDocument, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("fileName", entity.FileName),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert generated document", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *documenttemplate.GeneratedDocument,
) (*documenttemplate.GeneratedDocument, error) {
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
		Where("gdoc.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update generated document", zap.Error(rErr))
		return nil, rErr
	}

	if err = dberror.CheckRowsAffected(results, "Generated Document", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	entity *documenttemplate.GeneratedDocument,
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
		log.Error("failed to delete generated document", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "Generated Document", entity.ID.String())
}
