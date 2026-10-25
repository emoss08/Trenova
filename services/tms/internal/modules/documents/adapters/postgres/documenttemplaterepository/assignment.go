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
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var assignmentErrors = dberror.Mapping{Entity: "DocumentTemplateAssignment"}

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

func (r *assignmentRepository) List(
	ctx context.Context,
	req *documenttemplateports.ListDocumentTemplateAssignmentsRequest,
) (*pagination.ListResult[*documenttemplate.DocumentTemplateAssignment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*documenttemplate.DocumentTemplateAssignment], error) {
			a := documenttemplatedb.DocumentTemplateAssignments
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				a.Where(
					keel.When(req.Kind != "", a.Kind.Eq(req.Kind)),
					keel.When(
						req.CustomerID != nil && !req.CustomerID.IsNil(),
						a.CustomerID.Eq(derefID(req.CustomerID)),
					),
				),
				&listquery.Request[documenttemplate.DocumentTemplateAssignment]{
					Filter: req.Filter,
					Shape: func(
						q *keel.Query[documenttemplate.DocumentTemplateAssignment],
					) *keel.Query[documenttemplate.DocumentTemplateAssignment] {
						return q.Load(a.Customer(), a.Template())
					},
					Then: []keel.Order{a.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list document template assignments", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *assignmentRepository) ListConnection(
	ctx context.Context,
	req *documenttemplateports.ListDocumentTemplateAssignmentConnectionRequest,
) (*pagination.CursorListResult[*documenttemplate.DocumentTemplateAssignment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*documenttemplate.DocumentTemplateAssignment], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				documenttemplatedb.DocumentTemplateAssignments.Query(),
				&listquery.Request[documenttemplate.DocumentTemplateAssignment]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.DocumentTemplateAssignmentColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list document template assignment connection", zap.Error(err))
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
			a := documenttemplatedb.DocumentTemplateAssignments
			entities, err := a.Where(a.CustomerID.Eq(req.CustomerID)).
				Load(a.Template()).
				Order(a.Kind.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to load customer template assignments", zap.Error(err))
				return nil, err
			}
			if entities == nil {
				entities = []*documenttemplate.DocumentTemplateAssignment{}
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
			db := r.db.Keel(ctx)
			a := documenttemplatedb.DocumentTemplateAssignments
			inserted, err := a.Upsert(
				ctx,
				db,
				[]*documenttemplate.DocumentTemplateAssignment{entity},
				keel.OnConflict(a.OrganizationID, a.BusinessUnitID, a.Kind, a.CustomerID).
					Update(a.TemplateID, a.AssignedByID),
			)
			if err != nil {
				r.l.Error(
					"failed to assign document template",
					zap.String("customerId", entity.CustomerID.String()),
					zap.String("kind", string(entity.Kind)),
					zap.Error(err),
				)
				return nil, assignmentErrors.Map(err, entity.CustomerID.String())
			}
			if len(inserted) == 1 && inserted[0] {
				return entity, nil
			}

			stored, err := a.Where(a.Kind.Eq(entity.Kind), a.CustomerID.Eq(entity.CustomerID)).
				One(ctx, db)
			if err != nil {
				return nil, assignmentErrors.Map(err, entity.CustomerID.String())
			}
			*entity = *stored

			return entity, nil
		},
	)
}

func (r *assignmentRepository) Unassign(
	ctx context.Context,
	req *documenttemplateports.UnassignDocumentTemplateRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		a := documenttemplatedb.DocumentTemplateAssignments
		n, err := a.Where(a.CustomerID.Eq(req.CustomerID), a.Kind.Eq(req.Kind)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to unassign document template",
				zap.String("customerId", req.CustomerID.String()),
				zap.String("kind", string(req.Kind)),
				zap.Error(err),
			)
			return assignmentErrors.Map(err, req.CustomerID.String())
		}

		return dberror.CheckAffected(n, "DocumentTemplateAssignment", req.CustomerID.String())
	})
}
