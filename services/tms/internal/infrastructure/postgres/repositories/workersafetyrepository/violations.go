package workersafetyrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (r *repository) ListViolations(
	ctx context.Context,
	req *repositories.ListWorkerSafetyViolationsRequest,
) ([]*worker.WorkerSafetyViolation, error) {
	cols := buncolgen.WorkerSafetyViolationColumns
	entities := make([]*worker.WorkerSafetyViolation, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerSafetyViolationScopeTenant(sq, req.TenantInfo)
			if !req.SafetyEventID.IsNil() {
				sq = sq.Where(cols.SafetyEventID.Eq(), req.SafetyEventID)
			}
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			return sq
		}).
		Order(cols.Basic.OrderAsc()).
		Order(cols.SeverityWeight.OrderDesc())

	// The date lives on the event, so a window filter has to reach through it.
	if req.Since > 0 {
		q = q.Join("JOIN worker_safety_events AS wsev").
			JoinOn("wsev.id = ?", bun.Ident(cols.SafetyEventID.String())).
			JoinOn("wsev.organization_id = ?", bun.Ident(cols.OrganizationID.String())).
			JoinOn("wsev.business_unit_id = ?", bun.Ident(cols.BusinessUnitID.String())).
			JoinOn("wsev.occurred_at >= ?", req.Since)
	}
	if req.IncludeEvent {
		q = q.Relation(buncolgen.WorkerSafetyViolationRelations.SafetyEvent)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list worker safety violations", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetViolationByID(
	ctx context.Context,
	req *repositories.GetWorkerSafetyViolationByIDRequest,
) (*worker.WorkerSafetyViolation, error) {
	entity := new(worker.WorkerSafetyViolation)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerSafetyViolationScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerSafetyViolationColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerSafetyViolation")
	}

	return entity, nil
}

func (r *repository) CreateViolation(
	ctx context.Context,
	entity *worker.WorkerSafetyViolation,
) (*worker.WorkerSafetyViolation, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create worker safety violation", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateViolation(
	ctx context.Context,
	entity *worker.WorkerSafetyViolation,
) (*worker.WorkerSafetyViolation, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerSafetyViolationColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update worker safety violation", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(
		results,
		"WorkerSafetyViolation",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) DeleteViolation(
	ctx context.Context,
	req *repositories.GetWorkerSafetyViolationByIDRequest,
) error {
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*worker.WorkerSafetyViolation)(nil)).
		Apply(func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.WorkerSafetyViolationScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.WorkerSafetyViolationColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete worker safety violation", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "WorkerSafetyViolation", req.ID.String())
}
