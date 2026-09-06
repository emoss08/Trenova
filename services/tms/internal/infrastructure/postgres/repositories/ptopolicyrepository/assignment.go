package ptopolicyrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const defaultOpenAssignmentPageSize = 100

func policyWithRules(sq *bun.SelectQuery) *bun.SelectQuery {
	return sq.Relation(buncolgen.PTOPolicyRelations.Rules, orderRules)
}

func (r *repository) ListAssignments(
	ctx context.Context,
	req *repositories.ListPTOAssignmentsRequest,
) ([]*worker.WorkerPTOPolicyAssignment, error) {
	cols := buncolgen.WorkerPTOPolicyAssignmentColumns
	entities := make([]*worker.WorkerPTOPolicyAssignment, 0, 4)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPTOPolicyAssignmentScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID)
		}).
		Order(cols.EffectiveFrom.OrderDesc())
	if req.IncludePolicy {
		q = q.Relation(buncolgen.WorkerPTOPolicyAssignmentRelations.Policy, policyWithRules)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list PTO assignments", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetAssignmentByID(
	ctx context.Context,
	req *repositories.GetPTOAssignmentByIDRequest,
) (*worker.WorkerPTOPolicyAssignment, error) {
	entity := new(worker.WorkerPTOPolicyAssignment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.WorkerPTOPolicyAssignmentRelations.Policy, policyWithRules).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPTOPolicyAssignmentScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerPTOPolicyAssignmentColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerPTOPolicyAssignment")
	}

	return entity, nil
}

func (r *repository) GetActiveAssignment(
	ctx context.Context,
	req *repositories.GetPTOAssignmentRequest,
) (*worker.WorkerPTOPolicyAssignment, error) {
	cols := buncolgen.WorkerPTOPolicyAssignmentColumns
	entity := new(worker.WorkerPTOPolicyAssignment)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerPTOPolicyAssignmentScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID)
			if req.AsOf > 0 {
				sq = sq.Where(cols.EffectiveFrom.Lte(), req.AsOf).
					WhereGroup(" AND ", func(inner *bun.SelectQuery) *bun.SelectQuery {
						return inner.Where(cols.EffectiveTo.IsNull()).
							WhereOr(cols.EffectiveTo.Gt(), req.AsOf)
					})
			} else {
				sq = sq.Where(cols.EffectiveTo.IsNull())
			}
			return sq
		}).
		Order(cols.EffectiveFrom.OrderDesc()).
		Limit(1)
	if req.IncludePolicy {
		q = q.Relation(buncolgen.WorkerPTOPolicyAssignmentRelations.Policy, policyWithRules)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerPTOPolicyAssignment")
	}

	return entity, nil
}

func (r *repository) CreateAssignment(
	ctx context.Context,
	entity *worker.WorkerPTOPolicyAssignment,
) (*worker.WorkerPTOPolicyAssignment, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create PTO assignment", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateAssignment(
	ctx context.Context,
	entity *worker.WorkerPTOPolicyAssignment,
) (*worker.WorkerPTOPolicyAssignment, error) {
	ov := entity.Version
	entity.Version++

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerPTOPolicyAssignmentColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update PTO assignment", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(res, "WorkerPTOPolicyAssignment", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListOpenAssignments(
	ctx context.Context,
	req *repositories.ListOpenPTOAssignmentsRequest,
) ([]*worker.WorkerPTOPolicyAssignment, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultOpenAssignmentPageSize
	}

	cols := buncolgen.WorkerPTOPolicyAssignmentColumns
	entities := make([]*worker.WorkerPTOPolicyAssignment, 0, limit)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerPTOPolicyAssignmentScopeTenant(sq, req.TenantInfo).
				Where(cols.EffectiveTo.IsNull())
			if !req.PolicyID.IsNil() {
				sq = sq.Where(cols.PTOPolicyID.Eq(), req.PolicyID)
			}
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if !req.AfterID.IsNil() {
				sq = sq.Where(cols.ID.Gt(), req.AfterID)
			}
			return sq
		}).
		Order(cols.ID.OrderAsc()).
		Limit(limit)
	if req.IncludePolicy {
		q = q.Relation(buncolgen.WorkerPTOPolicyAssignmentRelations.Policy, policyWithRules)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list open PTO assignments", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

type tenantRow struct {
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func (r *repository) ListTenantsWithOpenAssignments(
	ctx context.Context,
) ([]pagination.TenantInfo, error) {
	cols := buncolgen.WorkerPTOPolicyAssignmentColumns
	rows := make([]tenantRow, 0, 16)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerPTOPolicyAssignment)(nil)).
		ColumnExpr(cols.OrganizationID.String()).
		ColumnExpr(cols.BusinessUnitID.String()).
		Where(cols.EffectiveTo.IsNull()).
		GroupExpr(cols.OrganizationID.String()+", "+cols.BusinessUnitID.String()).
		Scan(ctx, &rows)
	if err != nil {
		r.l.Error("failed to list tenants with open PTO assignments", zap.Error(err))
		return nil, err
	}

	tenants := make([]pagination.TenantInfo, 0, len(rows))
	for _, row := range rows {
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: row.OrganizationID,
			BuID:  row.BusinessUnitID,
		})
	}

	return tenants, nil
}
