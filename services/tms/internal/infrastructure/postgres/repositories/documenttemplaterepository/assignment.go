package documenttemplaterepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
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
) repositories.DocumentTemplateAssignmentRepository {
	return &assignmentRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-template-assignment-repository"),
	}
}

func (r *assignmentRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDocumentTemplateAssignmentsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.DocumentTemplateAssignmentTable.Alias,
		req.Filter,
		(*documenttemplate.DocumentTemplateAssignment)(nil),
	)

	cols := buncolgen.DocumentTemplateAssignmentColumns

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
	req *repositories.ListDocumentTemplateAssignmentsRequest,
) (*pagination.ListResult[*documenttemplate.DocumentTemplateAssignment], error) {
	log := r.l.With(zap.String("operation", "List"))

	rel := buncolgen.DocumentTemplateAssignmentRelations
	entities := make(
		[]*documenttemplate.DocumentTemplateAssignment, 0, req.Filter.Pagination.SafeLimit(),
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
}

func (r *assignmentRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListDocumentTemplateAssignmentConnectionRequest,
) (*pagination.CursorListResult[*documenttemplate.DocumentTemplateAssignment], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*documenttemplate.DocumentTemplateAssignment)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				buncolgen.DocumentTemplateAssignmentTable.Alias,
				req.Filter,
				(*documenttemplate.DocumentTemplateAssignment)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count document template assignments", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*documenttemplate.DocumentTemplateAssignment]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(
				entities *[]*documenttemplate.DocumentTemplateAssignment,
			) *bun.SelectQuery {
				q := dba.NewSelect().Model(entities)
				if len(req.DocumentTemplateAssignmentColumns) == 0 {
					q = q.ColumnExpr(buncolgen.DocumentTemplateAssignmentTable.All())
				} else {
					q = q.Column(req.DocumentTemplateAssignmentColumns...)
				}
				return q
			},
			Apply: func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					q,
					buncolgen.DocumentTemplateAssignmentTable.Alias,
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
}

// GetForCustomer returns every kind this customer overrides, which is what the
// customer detail tab shows without having to ask per kind.
func (r *assignmentRepository) GetForCustomer(
	ctx context.Context,
	req *repositories.GetCustomerAssignmentsRequest,
) ([]*documenttemplate.DocumentTemplateAssignment, error) {
	log := r.l.With(
		zap.String("operation", "GetForCustomer"),
		zap.String("customerId", req.CustomerID.String()),
	)

	cols := buncolgen.DocumentTemplateAssignmentColumns
	rel := buncolgen.DocumentTemplateAssignmentRelations
	entities := make([]*documenttemplate.DocumentTemplateAssignment, 0, 8)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(rel.Template).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentTemplateAssignmentScopeTenant(sq, req.TenantInfo).
				Where(cols.CustomerID.Eq(), req.CustomerID)
		}).
		Order(cols.Kind.OrderAsc()).
		Scan(ctx); err != nil {
		log.Error("failed to load customer template assignments", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

// Assign upserts on the (kind, customer) scope the unique index already enforces.
// Re-assigning replaces rather than accumulating, so resolution never has to
// break a tie between two rows.
func (r *assignmentRepository) Assign(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplateAssignment,
) (*documenttemplate.DocumentTemplateAssignment, error) {
	log := r.l.With(
		zap.String("operation", "Assign"),
		zap.String("customerId", entity.CustomerID.String()),
		zap.String("kind", string(entity.Kind)),
	)

	cols := buncolgen.DocumentTemplateAssignmentColumns

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
}

func (r *assignmentRepository) Unassign(
	ctx context.Context,
	req *repositories.UnassignDocumentTemplateRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Unassign"),
		zap.String("customerId", req.CustomerID.String()),
		zap.String("kind", string(req.Kind)),
	)

	cols := buncolgen.DocumentTemplateAssignmentColumns

	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*documenttemplate.DocumentTemplateAssignment)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.DocumentTemplateAssignmentScopeTenantDelete(dq, req.TenantInfo).
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
}
