package documenttemplaterepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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
		l:  p.Logger.Named("postgres.document-template-repository"),
	}
}

// orderVersions puts the newest revision first, which is the order the version
// rail reads in and the order a rollback picker needs.
func orderVersions(sq *bun.SelectQuery) *bun.SelectQuery {
	return sq.Order(buncolgen.DocumentTemplateVersionColumns.VersionNumber.OrderDesc())
}

func (r *templateRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDocumentTemplatesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.DocumentTemplateTable.Alias,
		req.Filter,
		(*documenttemplate.DocumentTemplate)(nil),
	)

	if req.Kind != "" {
		q = q.Where(buncolgen.DocumentTemplateColumns.Kind.Eq(), req.Kind)
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *templateRepository) List(
	ctx context.Context,
	req *repositories.ListDocumentTemplatesRequest,
) (*pagination.ListResult[*documenttemplate.DocumentTemplate], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*documenttemplate.DocumentTemplate, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.DocumentTemplateRelations.ActiveVersion).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count document templates", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*documenttemplate.DocumentTemplate]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *templateRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListDocumentTemplateConnectionRequest,
) (*pagination.CursorListResult[*documenttemplate.DocumentTemplate], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*documenttemplate.DocumentTemplate)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				buncolgen.DocumentTemplateTable.Alias,
				req.Filter,
				(*documenttemplate.DocumentTemplate)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count document templates", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*documenttemplate.DocumentTemplate]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*documenttemplate.DocumentTemplate) *bun.SelectQuery {
				q := dba.NewSelect().Model(entities)
				if len(req.DocumentTemplateColumns) == 0 {
					q = q.ColumnExpr(buncolgen.DocumentTemplateTable.All())
				} else {
					q = q.Column(req.DocumentTemplateColumns...)
				}
				return q
			},
			Apply: func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					q,
					buncolgen.DocumentTemplateTable.Alias,
					req.Filter,
					req.Cursor,
					(*documenttemplate.DocumentTemplate)(nil),
				)
			},
		},
	)
	if err != nil {
		log.Error("failed to cursor list document templates", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *templateRepository) GetByID(
	ctx context.Context,
	req *repositories.GetDocumentTemplateByIDRequest,
) (*documenttemplate.DocumentTemplate, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.DocumentTemplateID.String()),
	)

	cols := buncolgen.DocumentTemplateColumns
	rel := buncolgen.DocumentTemplateRelations
	entity := new(documenttemplate.DocumentTemplate)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentTemplateScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.DocumentTemplateID)
		})

	if req.IncludeActiveVersion {
		q = q.Relation(rel.ActiveVersion)
	}

	if req.IncludeVersions {
		q = q.Relation(rel.Versions, orderVersions)
	}

	if req.IncludeAssignments {
		q = q.Relation(rel.Assignments).
			Relation(buncolgen.Rel(
				rel.Assignments,
				buncolgen.DocumentTemplateAssignmentRelations.Customer,
			))
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get document template", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DocumentTemplate")
	}

	return entity, nil
}

func (r *templateRepository) Create(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplate,
) (*documenttemplate.DocumentTemplate, error) {
	log := r.l.With(zap.String("operation", "Create"))

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to create document template", zap.Error(err))
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
		zap.String("id", entity.ID.String()),
	)

	cols := buncolgen.DocumentTemplateColumns
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update document template", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(
		results, "DocumentTemplate", entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

// Delete removes a template and, by cascade, its versions and assignments. A
// version that produced a generated document is protected by RESTRICT on
// generated_documents, so the delete fails rather than orphaning the record of
// what a customer was sent.
func (r *templateRepository) Delete(
	ctx context.Context,
	req *repositories.GetDocumentTemplateByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.DocumentTemplateID.String()),
	)

	cols := buncolgen.DocumentTemplateColumns

	// The template points at its active version and the version points back, so
	// the pointer has to be dropped before the cascade can remove the versions.
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model((*documenttemplate.DocumentTemplate)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.DocumentTemplateScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.DocumentTemplateID)
			}).
			Set(cols.ActiveVersionID.SetNull()).
			Exec(c); uErr != nil {
			return uErr
		}

		results, dErr := r.db.DBForContext(c).
			NewDelete().
			Model((*documenttemplate.DocumentTemplate)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.DocumentTemplateScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ID.Eq(), req.DocumentTemplateID)
			}).
			Exec(c)
		if dErr != nil {
			return dErr
		}

		return dberror.CheckRowsAffected(
			results, "DocumentTemplate", req.DocumentTemplateID.String(),
		)
	})
	if err != nil {
		log.Error("failed to delete document template", zap.Error(err))
		return err
	}

	return nil
}

func (r *templateRepository) SelectOptions(
	ctx context.Context,
	req *repositories.DocumentTemplateSelectOptionsRequest,
) (*pagination.ListResult[*documenttemplate.DocumentTemplate], error) {
	cols := buncolgen.DocumentTemplateColumns

	return dbhelper.SelectOptions[*documenttemplate.DocumentTemplate](
		ctx,
		r.db.DBForContext(ctx),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.Code,
				cols.Name,
				cols.Description,
				cols.Kind,
				cols.IsOrgDefault,
			},
			OrgColumnRef: &cols.OrganizationID,
			BuColumnRef:  &cols.BusinessUnitID,
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				if req.Kind != "" {
					q = q.Where(cols.Kind.Eq(), req.Kind)
				}
				// Only templates that render anything are offerable: assigning a
				// customer to a draft-only template would silently fall through to
				// the built-in.
				return q.Where(cols.ActiveVersionID.IsNotNull()).
					Order(cols.Name.OrderAsc())
			},
			EntityName: "DocumentTemplate",
			SearchColumnRefs: []buncolgen.Column{
				cols.Code,
				cols.Name,
				cols.Description,
			},
		},
	)
}
