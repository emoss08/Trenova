package workerdqfrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultVerificationPageSize = 200
	defaultRetentionPageSize    = 200
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

func New(p Params) repositories.WorkerDQFRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-dqf-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

func (r *repository) ListVerifications(
	ctx context.Context,
	req *repositories.ListEmploymentVerificationsRequest,
) ([]*worker.WorkerEmploymentVerification, error) {
	cols := buncolgen.WorkerEmploymentVerificationColumns
	entities := make([]*worker.WorkerEmploymentVerification, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerEmploymentVerificationScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if req.OutstandingOnly {
				sq = sq.Where(
					cols.Status.In(),
					bun.In([]worker.EmploymentVerificationStatus{
						worker.VerificationPending,
						worker.VerificationRequested,
					}),
				)
			}
			return sq
		}).
		OrderExpr(cols.EmployedTo.Qualified() + " DESC NULLS LAST").
		Order(cols.CreatedAt.OrderDesc()).
		Limit(limitOr(req.Limit, defaultVerificationPageSize))

	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerEmploymentVerificationRelations.Worker)
	}
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerEmploymentVerificationRelations.Document)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list employment verifications", zap.Error(err))
		return nil, fmt.Errorf("list employment verifications: %w", err)
	}

	return entities, nil
}

func (r *repository) ListVerificationsByWorkerIDs(
	ctx context.Context,
	req *repositories.ListEmploymentVerificationsByWorkerIDsRequest,
) (map[pulid.ID][]*worker.WorkerEmploymentVerification, error) {
	cols := buncolgen.WorkerEmploymentVerificationColumns
	entities := make([]*worker.WorkerEmploymentVerification, 0, len(req.WorkerIDs)*8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerEmploymentVerificationScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.In(), bun.List(req.WorkerIDs))
		}).
		OrderExpr(cols.EmployedTo.Qualified() + " DESC NULLS LAST").
		Order(cols.CreatedAt.OrderDesc()).
		Limit(len(req.WorkerIDs) * defaultVerificationPageSize)

	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerEmploymentVerificationRelations.Worker)
	}
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerEmploymentVerificationRelations.Document)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list employment verifications by workers", zap.Error(err))
		return nil, fmt.Errorf("list employment verifications by workers: %w", err)
	}

	return sliceutils.GroupBy(entities, func(v *worker.WorkerEmploymentVerification) pulid.ID {
		return v.WorkerID
	}), nil
}

func (r *repository) GetVerificationByID(
	ctx context.Context,
	req *repositories.GetEmploymentVerificationByIDRequest,
) (*worker.WorkerEmploymentVerification, error) {
	entity := new(worker.WorkerEmploymentVerification)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerEmploymentVerificationScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerEmploymentVerificationColumns.ID.Eq(), req.ID)
		})
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerEmploymentVerificationRelations.Document)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerEmploymentVerification")
	}

	return entity, nil
}

func (r *repository) CreateVerification(
	ctx context.Context,
	entity *worker.WorkerEmploymentVerification,
) (*worker.WorkerEmploymentVerification, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create employment verification", zap.Error(err))
		return nil, fmt.Errorf("create employment verification: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateVerification(
	ctx context.Context,
	entity *worker.WorkerEmploymentVerification,
) (*worker.WorkerEmploymentVerification, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerEmploymentVerificationColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update employment verification", zap.Error(err))
		return nil, fmt.Errorf("update employment verification: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		results,
		"WorkerEmploymentVerification",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) DeleteVerification(
	ctx context.Context,
	req *repositories.GetEmploymentVerificationByIDRequest,
) error {
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*worker.WorkerEmploymentVerification)(nil)).
		Apply(func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.WorkerEmploymentVerificationScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.WorkerEmploymentVerificationColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete employment verification", zap.Error(err))
		return fmt.Errorf("delete employment verification: %w", err)
	}

	return dberror.CheckRowsAffected(
		results,
		"WorkerEmploymentVerification",
		req.ID.String(),
	)
}

// ListRetentionCandidates finds terminated drivers whose file has passed its
// hold period. The window is applied in SQL so the caller reads the handful
// that qualify rather than every worker who ever left.
func (r *repository) ListRetentionCandidates(
	ctx context.Context,
	req *repositories.ListDQFRetentionCandidatesRequest,
) ([]repositories.DQFRetentionCandidate, error) {
	workerCols := buncolgen.WorkerColumns
	profileCols := buncolgen.WorkerProfileColumns

	candidates := make([]repositories.DQFRetentionCandidate, 0, 32)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.Worker)(nil)).
		ColumnExpr("wrk.id AS worker_id").
		ColumnExpr("wrk.organization_id AS organization_id").
		ColumnExpr("wrk.business_unit_id AS business_unit_id").
		ColumnExpr("wrk.first_name AS first_name").
		ColumnExpr("wrk.last_name AS last_name").
		ColumnExpr("wrkp.termination_date AS termination_date").
		Join("JOIN worker_profiles AS wrkp ON wrkp.worker_id = wrk.id"+
			" AND wrkp.organization_id = wrk.organization_id"+
			" AND wrkp.business_unit_id = wrk.business_unit_id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerScopeTenant(sq, req.TenantInfo).
				Where(profileCols.TerminationDate.IsNotNull()).
				Where(
					profileCols.TerminationDate.Qualified()+" + ? <= ?",
					req.RetentionSeconds,
					req.AsOf,
				)
		}).
		Order(workerCols.ID.OrderAsc()).
		Limit(limitOr(req.Limit, defaultRetentionPageSize)).
		Scan(ctx, &candidates)
	if err != nil {
		r.l.Error("failed to list dqf retention candidates", zap.Error(err))
		return nil, fmt.Errorf("list dqf retention candidates: %w", err)
	}

	return candidates, nil
}
