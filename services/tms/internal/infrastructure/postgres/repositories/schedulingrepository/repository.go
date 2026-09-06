package schedulingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultTemplatePageSize   = 200
	defaultAssignmentPageSize = 200
	defaultSwapPageSize       = 200
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

func New(p Params) repositories.SchedulingRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.scheduling-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

func (r *repository) ListTemplates(
	ctx context.Context,
	req *repositories.ListShiftTemplatesRequest,
) ([]*worker.ShiftTemplate, error) {
	cols := buncolgen.ShiftTemplateColumns
	entities := make([]*worker.ShiftTemplate, 0, 8)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.ShiftTemplateScopeTenant(sq, req.TenantInfo)
			if req.ActiveOnly {
				sq = sq.Where(cols.Status.Eq(), domaintypes.StatusActive)
			}
			return sq
		}).
		Order(cols.Name.OrderAsc()).
		Limit(limitOr(req.Limit, defaultTemplatePageSize)).
		Scan(ctx); err != nil {
		r.l.Error("failed to list shift templates", zap.Error(err))
		return nil, fmt.Errorf("list shift templates: %w", err)
	}

	return entities, nil
}

func (r *repository) GetTemplateByID(
	ctx context.Context,
	req *repositories.GetShiftTemplateByIDRequest,
) (*worker.ShiftTemplate, error) {
	entity := new(worker.ShiftTemplate)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShiftTemplateScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ShiftTemplateColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "ShiftTemplate")
	}

	return entity, nil
}

func (r *repository) CreateTemplate(
	ctx context.Context,
	entity *worker.ShiftTemplate,
) (*worker.ShiftTemplate, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create shift template", zap.Error(err))
		return nil, fmt.Errorf("create shift template: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateTemplate(
	ctx context.Context,
	entity *worker.ShiftTemplate,
) (*worker.ShiftTemplate, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.ShiftTemplateColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update shift template", zap.Error(err))
		return nil, fmt.Errorf("update shift template: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "ShiftTemplate", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) CountTemplateAssignments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	templateID pulid.ID,
) (int, error) {
	cols := buncolgen.WorkerShiftAssignmentColumns
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerShiftAssignment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerShiftAssignmentScopeTenant(sq, tenantInfo).
				Where(cols.ShiftTemplateID.Eq(), templateID).
				Where(cols.EffectiveTo.IsNull())
		}).
		Count(ctx)
	if err != nil {
		r.l.Error("failed to count template assignments", zap.Error(err))
		return 0, fmt.Errorf("count template assignments: %w", err)
	}

	return total, nil
}

func (r *repository) CountTemplateAssignmentsByIDs(
	ctx context.Context,
	req *repositories.CountShiftTemplateAssignmentsRequest,
) (map[pulid.ID]int, error) {
	if len(req.TemplateIDs) == 0 {
		return map[pulid.ID]int{}, nil
	}

	cols := buncolgen.WorkerShiftAssignmentColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerShiftAssignment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerShiftAssignmentScopeTenant(sq, req.TenantInfo).
				Where(cols.ShiftTemplateID.In(), bun.In(req.TemplateIDs)).
				Where(cols.EffectiveTo.IsNull())
		})

	counts, err := dbhelper.CountByID(ctx, q, cols.ShiftTemplateID, len(req.TemplateIDs))
	if err != nil {
		r.l.Error("failed to count template assignments by template", zap.Error(err))
		return nil, fmt.Errorf("count template assignments by template: %w", err)
	}

	return counts, nil
}

func (r *repository) ListAssignments(
	ctx context.Context,
	req *repositories.ListShiftAssignmentsRequest,
) ([]*worker.WorkerShiftAssignment, error) {
	cols := buncolgen.WorkerShiftAssignmentColumns
	entities := make([]*worker.WorkerShiftAssignment, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerShiftAssignmentScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if req.ActiveAt > 0 {
				sq = sq.
					Where(cols.EffectiveFrom.Lte(), req.ActiveAt).
					WhereGroup(" AND ", func(eq *bun.SelectQuery) *bun.SelectQuery {
						return eq.
							Where(cols.EffectiveTo.IsNull()).
							WhereOr(cols.EffectiveTo.Gte(), req.ActiveAt)
					})
			}
			return sq
		}).
		Order(cols.EffectiveFrom.OrderDesc()).
		Limit(limitOr(req.Limit, defaultAssignmentPageSize))

	if req.IncludeTemplate {
		q = q.Relation(buncolgen.WorkerShiftAssignmentRelations.ShiftTemplate)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list shift assignments", zap.Error(err))
		return nil, fmt.Errorf("list shift assignments: %w", err)
	}

	return entities, nil
}

func (r *repository) GetAssignmentByID(
	ctx context.Context,
	req *repositories.GetShiftAssignmentByIDRequest,
) (*worker.WorkerShiftAssignment, error) {
	entity := new(worker.WorkerShiftAssignment)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerShiftAssignmentScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerShiftAssignmentColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerShiftAssignment")
	}

	return entity, nil
}

func (r *repository) CreateAssignment(
	ctx context.Context,
	entity *worker.WorkerShiftAssignment,
) (*worker.WorkerShiftAssignment, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create shift assignment", zap.Error(err))
		return nil, fmt.Errorf("create shift assignment: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateAssignment(
	ctx context.Context,
	entity *worker.WorkerShiftAssignment,
) (*worker.WorkerShiftAssignment, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerShiftAssignmentColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update shift assignment", zap.Error(err))
		return nil, fmt.Errorf("update shift assignment: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		results,
		"WorkerShiftAssignment",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListPreferences(
	ctx context.Context,
	req *repositories.ListAvailabilityPreferencesRequest,
) ([]*worker.WorkerAvailabilityPreference, error) {
	cols := buncolgen.WorkerAvailabilityPreferenceColumns
	entities := make([]*worker.WorkerAvailabilityPreference, 0, 7)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerAvailabilityPreferenceScopeTenant(sq, req.TenantInfo)
			// An empty worker reads the whole roster's stated availability,
			// which is what the rota needs: seven rows a worker at most, so
			// one read beats one per line on the board.
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			return sq
		}).
		Order(cols.WorkerID.OrderAsc(), cols.DayOfWeek.OrderAsc()).
		Scan(ctx); err != nil {
		r.l.Error("failed to list availability preferences", zap.Error(err))
		return nil, fmt.Errorf("list availability preferences: %w", err)
	}

	return entities, nil
}

// UpsertPreference writes one weekday's statement, replacing whatever was
// there. A worker has exactly one statement per weekday, so a collision is a
// correction rather than a second opinion.
func (r *repository) UpsertPreference(
	ctx context.Context,
	entity *worker.WorkerAvailabilityPreference,
) (*worker.WorkerAvailabilityPreference, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On("CONFLICT (organization_id, business_unit_id, worker_id, day_of_week) DO UPDATE").
		Set("preference = EXCLUDED.preference").
		Set("note = EXCLUDED.note").
		Set("updated_at = EXCLUDED.updated_at").
		Set("version = worker_availability_preferences.version + 1").
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to upsert availability preference", zap.Error(err))
		return nil, fmt.Errorf("upsert availability preference: %w", err)
	}

	return entity, nil
}

func (r *repository) ListSwaps(
	ctx context.Context,
	req *repositories.ListShiftSwapsRequest,
) ([]*worker.ShiftSwapRequest, error) {
	cols := buncolgen.ShiftSwapRequestColumns
	entities := make([]*worker.ShiftSwapRequest, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.ShiftSwapRequestScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.WhereGroup(" AND ", func(wq *bun.SelectQuery) *bun.SelectQuery {
					return wq.
						Where(cols.RequestingWorkerID.Eq(), req.WorkerID).
						WhereOr(cols.CounterpartyWorkerID.Eq(), req.WorkerID)
				})
			}
			if req.OpenOnly {
				sq = sq.Where(cols.Status.In(), bun.In([]worker.ShiftSwapStatus{
					worker.SwapProposed,
					worker.SwapAccepted,
				}))
			}
			if req.Since > 0 {
				sq = sq.Where(cols.ShiftDate.Gte(), req.Since)
			}
			return sq
		}).
		Order(cols.ShiftDate.OrderAsc()).
		Limit(limitOr(req.Limit, defaultSwapPageSize))

	if req.IncludeWorkers {
		q = q.Relation(buncolgen.ShiftSwapRequestRelations.RequestingWorker).
			Relation(buncolgen.ShiftSwapRequestRelations.CounterpartyWorker)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list shift swaps", zap.Error(err))
		return nil, fmt.Errorf("list shift swaps: %w", err)
	}

	return entities, nil
}

func (r *repository) GetSwapByID(
	ctx context.Context,
	req *repositories.GetShiftSwapByIDRequest,
) (*worker.ShiftSwapRequest, error) {
	entity := new(worker.ShiftSwapRequest)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShiftSwapRequestScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ShiftSwapRequestColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "ShiftSwapRequest")
	}

	return entity, nil
}

func (r *repository) CreateSwap(
	ctx context.Context,
	entity *worker.ShiftSwapRequest,
) (*worker.ShiftSwapRequest, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create shift swap", zap.Error(err))
		return nil, fmt.Errorf("create shift swap: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateSwap(
	ctx context.Context,
	entity *worker.ShiftSwapRequest,
) (*worker.ShiftSwapRequest, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.ShiftSwapRequestColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update shift swap", zap.Error(err))
		return nil, fmt.Errorf("update shift swap: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "ShiftSwapRequest", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
