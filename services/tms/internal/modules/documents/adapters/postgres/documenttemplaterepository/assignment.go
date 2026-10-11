package documenttemplaterepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/documenttemplatecol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type AssignmentParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type assignmentRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewAssignmentRepository(
	p AssignmentParams,
) documenttemplateports.DocumentTemplateAssignmentRepository {
	return &assignmentRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-template-assignment-repository"),
	}
}

func (r *assignmentRepository) filterQuery(
	q *bun.SelectQuery,
	req *documenttemplateports.ListDocumentTemplateAssignmentsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		documenttemplatecol.DocumentTemplateAssignmentTable.Alias,
		req.Filter,
		(*documenttemplate.DocumentTemplateAssignment)(nil),
	)

	cols := documenttemplatecol.DocumentTemplateAssignmentColumns

	if req.Kind != "" {
		q = q.Where(cols.Kind.Eq(), req.Kind)
	}

	if req.CustomerID != nil && !req.CustomerID.IsNil() {
		q = q.Where(cols.CustomerID.Eq(), *req.CustomerID)
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *assignmentRepository) List(
	ctx context.Context,
	req *documenttemplateports.ListDocumentTemplateAssignmentsRequest,
) (*pagination.ListResult[*documenttemplate.DocumentTemplateAssignment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*documenttemplate.DocumentTemplateAssignment], error) {
			log := r.l.With(zap.String("operation", "List"))

			rel := documenttemplatecol.DocumentTemplateAssignmentRelations
			entities := make(
				[]*documenttemplate.DocumentTemplateAssignment,
				0,
				req.Filter.Pagination.SafeLimit(),
			)

			total, err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Relation(rel.Customer).
				Relation(rel.Template).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return r.filterQuery(sq, req)
				}).
				ScanAndCount(ctx)
			if err != nil {
				log.Error("failed to scan and count document template assignments", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*documenttemplate.DocumentTemplateAssignment]{
				Items: entities,
				Total: total,
			}, nil
		},
	)
}

func (r *assignmentRepository) ListConnection(
	ctx context.Context,
	req *documenttemplateports.ListDocumentTemplateAssignmentConnectionRequest,
) (*pagination.CursorListResult[*documenttemplate.DocumentTemplateAssignment], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*documenttemplate.DocumentTemplateAssignment], error) {
			log := r.l.With(zap.String("operation", "ListConnection"))

			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.
					NewSelect().
					Model((*documenttemplate.DocumentTemplateAssignment)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return querybuilder.ApplyFiltersWithoutSort(
							sq,
							documenttemplatecol.DocumentTemplateAssignmentTable.Alias,
							req.Filter,
							(*documenttemplate.DocumentTemplateAssignment)(nil),
						)
					}).
					Count(ctx)
				if err != nil {
					log.Error("failed to count document template assignments", zap.Error(err))
					return nil, err
				}
				totalCount = &total
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*documenttemplate.DocumentTemplateAssignment]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(
						entities *[]*documenttemplate.DocumentTemplateAssignment,
					) *bun.SelectQuery {
						q := dba.NewSelect().Model(entities)
						if len(req.DocumentTemplateAssignmentColumns) == 0 {
							q = q.ColumnExpr(
								documenttemplatecol.DocumentTemplateAssignmentTable.All(),
							)
						} else {
							q = q.Column(req.DocumentTemplateAssignmentColumns...)
						}
						return q
					},
					Apply: func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
						return querybuilder.ApplyCursorFilters(
							q,
							documenttemplatecol.DocumentTemplateAssignmentTable.Alias,
							req.Filter,
							req.Cursor,
							(*documenttemplate.DocumentTemplateAssignment)(nil),
						)
					},
				},
			)
			if err != nil {
				log.Error("failed to cursor list document template assignments", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

// GetForCustomer returns every kind this customer overrides, which is what the
// customer detail tab shows without having to ask per kind.
func (r *assignmentRepository) GetForCustomer(
	ctx context.Context,
	req *documenttemplateports.GetCustomerAssignmentsRequest,
) ([]*documenttemplate.DocumentTemplateAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*documenttemplate.DocumentTemplateAssignment, error) {
			log := r.l.With(
				zap.String("operation", "GetForCustomer"),
				zap.String("customerId", req.CustomerID.String()),
			)

			cols := documenttemplatecol.DocumentTemplateAssignmentColumns
			rel := documenttemplatecol.DocumentTemplateAssignmentRelations
			entities := make([]*documenttemplate.DocumentTemplateAssignment, 0, 8)

			if err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Relation(rel.Template).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return documenttemplatecol.DocumentTemplateAssignmentScopeTenant(sq, req.TenantInfo).
						Where(cols.CustomerID.Eq(), req.CustomerID)
				}).
				Order(cols.Kind.OrderAsc()).
				Scan(ctx); err != nil {
				log.Error("failed to load customer template assignments", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

// Assign upserts on the (kind, customer) scope the unique index already enforces.
// Re-assigning replaces rather than accumulating, so resolution never has to
// break a tie between two rows.
func (r *assignmentRepository) Assign(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplateAssignment,
) (*documenttemplate.DocumentTemplateAssignment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documenttemplate.DocumentTemplateAssignment, error) {
			log := r.l.With(
				zap.String("operation", "Assign"),
				zap.String("customerId", entity.CustomerID.String()),
				zap.String("kind", string(entity.Kind)),
			)

			cols := documenttemplatecol.DocumentTemplateAssignmentColumns

			if _, err := r.db.DBForContext(ctx).
				NewInsert().
				Model(entity).
				On("CONFLICT (organization_id, business_unit_id, kind, customer_id) DO UPDATE").
				Set(cols.TemplateID.SetExcluded()).
				Set(cols.AssignedByID.SetExcluded()).
				Set(cols.UpdatedAt.SetExcluded()).
				// Qualified, not Inc: inside DO UPDATE both the target row and EXCLUDED
				// expose "version", and Postgres refuses the bare name as ambiguous.
				Set(cols.Version.SetExpr(cols.Version.Qualified() + " + 1")).
				Returning("*").
				Exec(ctx); err != nil {
				log.Error("failed to assign document template", zap.Error(err))
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *assignmentRepository) Unassign(
	ctx context.Context,
	req *documenttemplateports.UnassignDocumentTemplateRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		log := r.l.With(
			zap.String("operation", "Unassign"),
			zap.String("customerId", req.CustomerID.String()),
			zap.String("kind", string(req.Kind)),
		)

		cols := documenttemplatecol.DocumentTemplateAssignmentColumns

		results, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*documenttemplate.DocumentTemplateAssignment)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return documenttemplatecol.DocumentTemplateAssignmentScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.CustomerID.Eq(), req.CustomerID).
					Where(cols.Kind.Eq(), req.Kind)
			}).
			Exec(ctx)
		if err != nil {
			log.Error("failed to unassign document template", zap.Error(err))
			return err
		}

		return dberror.CheckRowsAffected(
			results, "DocumentTemplateAssignment", req.CustomerID.String(),
		)
	})
}
