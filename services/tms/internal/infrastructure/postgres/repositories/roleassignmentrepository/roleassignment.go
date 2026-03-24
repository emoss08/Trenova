package roleassignmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.RoleAssignmentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.role-assignment-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListRoleAssignmentsRequest,
) *bun.SelectQuery {
	if req.ExpandRoles {
		q = q.Relation("Role")
	}

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListRoleAssignmentsRequest,
) (*pagination.ListResult[*permission.UserRoleAssignment], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*permission.UserRoleAssignment, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count role assignments", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*permission.UserRoleAssignment]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetRoleAssignmentByIDRequest,
) (*permission.UserRoleAssignment, error) {
	entity := new(permission.UserRoleAssignment)
	q := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("ura.organization_id = ?", req.TenantInfo.OrgID).
				Where("ura.id = ?", req.RoleAssignmentID)
		})

	if req.ExpandRoles {
		q.Relation("Role")
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "RoleAssignment")
	}

	return entity, nil
}
