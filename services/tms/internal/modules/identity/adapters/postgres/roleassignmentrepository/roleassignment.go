package roleassignmentrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var roleAssignmentErrors = dberror.Mapping{Entity: "RoleAssignment"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) permissionports.RoleAssignmentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.role-assignment-repository"),
	}
}

func withRole(
	q *keel.Query[*rbac.UserRoleAssignment],
	expand bool,
) *keel.Query[*rbac.UserRoleAssignment] {
	if !expand {
		return q
	}

	return q.Load(permissiondb.UserRoleAssignments.Role())
}

func (r *repository) List(
	ctx context.Context,
	req *permissionports.ListRoleAssignmentsRequest,
) (*pagination.ListResult[*rbac.UserRoleAssignment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*rbac.UserRoleAssignment], error) {
			ura := permissiondb.UserRoleAssignments
			page := req.Filter.Pagination
			rows := withRole(ura.Query(), req.ExpandRoles).
				Limit(page.SafeLimit()).
				Offset(page.SafeOffset())

			var items []*rbac.UserRoleAssignment
			var total keel.RowCount
			if err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				keel.ListInto(rows, &items),
				keel.CountEstimateInto(ura.Query(), pagination.ExactTotalUpTo, &total),
			); err != nil {
				r.l.Error("failed to list role assignments", zap.Error(err))
				return nil, err
			}

			return listquery.Result(items, total), nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req permissionports.GetRoleAssignmentByIDRequest,
) (*rbac.UserRoleAssignment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*rbac.UserRoleAssignment, error) {
		ura := permissiondb.UserRoleAssignments
		entity, err := withRole(ura.Where(ura.ID.Eq(req.RoleAssignmentID)), req.ExpandRoles).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, roleAssignmentErrors.Map(err, req.RoleAssignmentID.String())
		}

		return entity, nil
	})
}
