package documenttemplaterepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate/documenttemplatedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const generatedVersionConstraint = "fk_generated_documents_template_version"

var templateErrors = dberror.Mapping{Entity: "DocumentTemplate"}

type TemplateParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type templateRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewTemplateRepository(p TemplateParams) documenttemplateports.DocumentTemplateRepository {
	return &templateRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-template-repository"),
	}
}

func (r *templateRepository) List(
	ctx context.Context,
	req *documenttemplateports.ListDocumentTemplatesRequest,
) (*pagination.ListResult[*documenttemplate.DocumentTemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*documenttemplate.DocumentTemplate], error) {
			t := documenttemplatedb.DocumentTemplates
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				t.Where(keel.When(req.Kind != "", t.Kind.Eq(req.Kind))),
				&listquery.Request[documenttemplate.DocumentTemplate]{
					Filter: req.Filter,
					Shape: func(
						q *keel.Query[*documenttemplate.DocumentTemplate],
					) *keel.Query[*documenttemplate.DocumentTemplate] {
						return q.Load(t.ActiveVersion())
					},
					Then: []keel.Order{t.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list document templates", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *templateRepository) ListConnection(
	ctx context.Context,
	req *documenttemplateports.ListDocumentTemplateConnectionRequest,
) (*pagination.CursorListResult[*documenttemplate.DocumentTemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*documenttemplate.DocumentTemplate], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				documenttemplatedb.DocumentTemplates.Query(),
				&listquery.Request[documenttemplate.DocumentTemplate]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.DocumentTemplateColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list document template connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *templateRepository) GetByID(
	ctx context.Context,
	req *documenttemplateports.GetDocumentTemplateByIDRequest,
) (*documenttemplate.DocumentTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*documenttemplate.DocumentTemplate, error) {
			t := documenttemplatedb.DocumentTemplates
			versions := t.Versions()
			assignments := t.Assignments()

			var loads []keel.Loader
			if req.IncludeActiveVersion {
				loads = append(loads, t.ActiveVersion())
			}
			if req.IncludeVersions {
				loads = append(loads, versions.Order(versions.VersionNumber.Desc()))
			}
			if req.IncludeAssignments {
				loads = append(loads, assignments.Load(assignments.Customer()))
			}

			entity, err := t.Where(t.ID.Eq(req.DocumentTemplateID)).
				Load(loads...).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get document template", zap.Error(err))
				return nil, templateErrors.Map(err, req.DocumentTemplateID.String())
			}

			return entity, nil
		},
	)
}

func (r *templateRepository) GetByIDs(
	ctx context.Context,
	req *documenttemplateports.GetDocumentTemplatesByIDsRequest,
) ([]*documenttemplate.DocumentTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*documenttemplate.DocumentTemplate, error) {
			if len(req.DocumentTemplateIDs) == 0 {
				return []*documenttemplate.DocumentTemplate{}, nil
			}

			t := documenttemplatedb.DocumentTemplates
			entities, err := t.Where(t.ID.In(req.DocumentTemplateIDs...)).
				Limit(len(req.DocumentTemplateIDs)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get document templates by ids", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *templateRepository) Create(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplate,
) (*documenttemplate.DocumentTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documenttemplate.DocumentTemplate, error) {
			err := documenttemplatedb.DocumentTemplates.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create document template", zap.Error(err))
				return nil, templateErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *templateRepository) Update(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplate,
) (*documenttemplate.DocumentTemplate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documenttemplate.DocumentTemplate, error) {
			err := documenttemplatedb.DocumentTemplates.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to update document template",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, templateErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

// Delete removes a template and, by cascade, its versions and assignments. A
// version that produced a generated document is protected by RESTRICT on
// generated_documents, so the delete fails rather than orphaning the record of
// what a customer was sent.
func (r *templateRepository) Delete(
	ctx context.Context,
	req *documenttemplateports.GetDocumentTemplateByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		t := documenttemplatedb.DocumentTemplates
		id := req.DocumentTemplateID.String()

		// The template points at its active version and the version points back, so
		// the pointer has to be dropped before the cascade can remove the versions.
		if _, err := t.Where(t.ID.Eq(req.DocumentTemplateID)).
			Update(ctx, db, t.ActiveVersionID.ToNull()); err != nil {
			r.l.Error("failed to delete document template", zap.Error(err))
			return templateErrors.Map(err, id)
		}

		n, err := t.Where(t.ID.Eq(req.DocumentTemplateID)).Delete(ctx, db)
		if err != nil {
			r.l.Error("failed to delete document template", zap.Error(err))
			return mapTemplateDeleteError(err, id)
		}

		return dberror.CheckAffected(n, "DocumentTemplate", id)
	})
}

func mapTemplateDeleteError(err error, id string) error {
	if dberror.IsForeignKeyConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == generatedVersionConstraint {
		return errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidReference,
			"This template produced documents that were sent, so its versions cannot be deleted",
		).WithInternal(err)
	}

	return templateErrors.Map(err, id)
}

func (r *templateRepository) SelectOptions(
	ctx context.Context,
	req *documenttemplateports.DocumentTemplateSelectOptionsRequest,
) (*pagination.ListResult[*documenttemplate.DocumentTemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*documenttemplate.DocumentTemplate], error) {
			t := documenttemplatedb.DocumentTemplates
			// Only templates that render anything are offerable: assigning a customer
			// to a draft-only template would silently fall through to the built-in.
			q := t.Pick(t.ID, t.Code, t.Name, t.Description, t.Kind, t.IsOrgDefault).
				Where(
					keel.When(req.Kind != "", t.Kind.Eq(req.Kind)),
					t.ActiveVersionID.IsNotNull(),
				).
				Order(t.Name.Asc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						t.Code.ContainsFold(s),
						t.Name.ContainsFold(s),
						t.Description.ContainsFold(s),
					)
				},
			)
		},
	)
}
