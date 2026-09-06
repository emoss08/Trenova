package orgstructurerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultPositionPageSize   = 200
	defaultDelegationPageSize = 200
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

func New(p Params) repositories.OrgStructureRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.org-structure-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

func (r *repository) ListPositions(
	ctx context.Context,
	req *repositories.ListJobPositionsRequest,
) ([]*worker.JobPosition, error) {
	cols := buncolgen.JobPositionColumns
	entities := make([]*worker.JobPosition, 0, 16)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.JobPositionScopeTenant(sq, req.TenantInfo)
			if req.ActiveOnly {
				sq = sq.Where(cols.Status.Eq(), domaintypes.StatusActive)
			}
			if req.DrivingOnly {
				sq = sq.Where(cols.IsDrivingPosition.Eq(), true)
			}
			if req.Department != "" {
				sq = sq.Where(cols.Department.Eq(), req.Department)
			}
			return sq
		}).
		Order(cols.Department.OrderAsc()).
		Order(cols.Title.OrderAsc()).
		Limit(limitOr(req.Limit, defaultPositionPageSize))

	if req.IncludeReportsTo {
		q = q.Relation(buncolgen.JobPositionRelations.ReportsTo)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list job positions", zap.Error(err))
		return nil, fmt.Errorf("list job positions: %w", err)
	}

	return entities, nil
}

func (r *repository) GetPositionByID(
	ctx context.Context,
	req *repositories.GetJobPositionByIDRequest,
) (*worker.JobPosition, error) {
	entity := new(worker.JobPosition)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.JobPositionScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.JobPositionColumns.ID.Eq(), req.ID)
		})
	if req.IncludeReportsTo {
		q = q.Relation(buncolgen.JobPositionRelations.ReportsTo)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "JobPosition")
	}

	return entity, nil
}

func (r *repository) CreatePosition(
	ctx context.Context,
	entity *worker.JobPosition,
) (*worker.JobPosition, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create job position", zap.Error(err))
		return nil, fmt.Errorf("create job position: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdatePosition(
	ctx context.Context,
	entity *worker.JobPosition,
) (*worker.JobPosition, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.JobPositionColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update job position", zap.Error(err))
		return nil, fmt.Errorf("update job position: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "JobPosition", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) CountPositionHolders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	positionID pulid.ID,
) (int, error) {
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.Worker)(nil)).
		Where("wrk.organization_id = ?", tenantInfo.OrgID).
		Where("wrk.business_unit_id = ?", tenantInfo.BuID).
		Where("wrk.position_id = ?", positionID).
		Count(ctx)
	if err != nil {
		r.l.Error("failed to count position holders", zap.Error(err))
		return 0, fmt.Errorf("count position holders: %w", err)
	}

	return total, nil
}

func (r *repository) ListDelegations(
	ctx context.Context,
	req *repositories.ListApprovalDelegationsRequest,
) ([]*worker.ApprovalDelegation, error) {
	cols := buncolgen.ApprovalDelegationColumns
	entities := make([]*worker.ApprovalDelegation, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.ApprovalDelegationScopeTenant(sq, req.TenantInfo)
			if !req.DelegatorID.IsNil() {
				sq = sq.Where(cols.DelegatorID.Eq(), req.DelegatorID)
			}
			if !req.DelegateID.IsNil() {
				sq = sq.Where(cols.DelegateID.Eq(), req.DelegateID)
			}
			if req.ActiveAt > 0 {
				sq = sq.
					Where(cols.StartsAt.Lte(), req.ActiveAt).
					WhereGroup(" AND ", func(eq *bun.SelectQuery) *bun.SelectQuery {
						return eq.
							Where(cols.EndsAt.IsNull()).
							WhereOr(cols.EndsAt.Gte(), req.ActiveAt)
					}).
					WhereGroup(" AND ", func(rq *bun.SelectQuery) *bun.SelectQuery {
						return rq.
							Where(cols.RevokedAt.IsNull()).
							WhereOr(cols.RevokedAt.Gt(), req.ActiveAt)
					})
			}
			return sq
		}).
		Order(cols.StartsAt.OrderDesc()).
		Limit(limitOr(req.Limit, defaultDelegationPageSize))

	if req.IncludeUsers {
		q = q.Relation(buncolgen.ApprovalDelegationRelations.Delegator).
			Relation(buncolgen.ApprovalDelegationRelations.Delegate)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list approval delegations", zap.Error(err))
		return nil, fmt.Errorf("list approval delegations: %w", err)
	}

	return entities, nil
}

func (r *repository) GetDelegationByID(
	ctx context.Context,
	req *repositories.GetApprovalDelegationByIDRequest,
) (*worker.ApprovalDelegation, error) {
	entity := new(worker.ApprovalDelegation)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ApprovalDelegationScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ApprovalDelegationColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "ApprovalDelegation")
	}

	return entity, nil
}

func (r *repository) CreateDelegation(
	ctx context.Context,
	entity *worker.ApprovalDelegation,
) (*worker.ApprovalDelegation, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create approval delegation", zap.Error(err))
		return nil, fmt.Errorf("create approval delegation: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateDelegation(
	ctx context.Context,
	entity *worker.ApprovalDelegation,
) (*worker.ApprovalDelegation, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.ApprovalDelegationColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update approval delegation", zap.Error(err))
		return nil, fmt.Errorf("update approval delegation: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		results,
		"ApprovalDelegation",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}
