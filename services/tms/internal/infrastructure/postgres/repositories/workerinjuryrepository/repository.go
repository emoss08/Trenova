package workerinjuryrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultInjuryPageSize  = 500
	defaultSummaryPageSize = 20
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

func New(p Params) repositories.WorkerInjuryRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-injury-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

func (r *repository) ListInjuries(
	ctx context.Context,
	req *repositories.ListWorkerInjuriesRequest,
) ([]*worker.WorkerInjury, error) {
	cols := buncolgen.WorkerInjuryColumns
	entities := make([]*worker.WorkerInjury, 0, 16)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerInjuryScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if req.CaseYear > 0 {
				sq = sq.Where(cols.CaseYear.Eq(), req.CaseYear)
			}
			if req.RecordableOnly {
				sq = sq.Where(
					cols.Classification.In(),
					bun.In([]worker.OSHACaseClassification{
						worker.CaseOtherRecordable,
						worker.CaseJobTransferOrRestriction,
						worker.CaseDaysAway,
						worker.CaseDeath,
					}),
				)
			}
			if req.OpenOnly {
				sq = sq.Where(cols.Status.Eq(), worker.InjuryCaseOpen)
			}
			return sq
		}).
		Order(cols.CaseYear.OrderDesc()).
		Order(cols.CaseNumber.OrderDesc()).
		Limit(limitOr(req.Limit, defaultInjuryPageSize))

	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerInjuryRelations.Worker)
	}
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerInjuryRelations.Document)
	}
	if req.IncludeEvent {
		q = q.Relation(buncolgen.WorkerInjuryRelations.SafetyEvent)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list injuries", zap.Error(err))
		return nil, fmt.Errorf("list injuries: %w", err)
	}

	return entities, nil
}

func (r *repository) GetInjuryByID(
	ctx context.Context,
	req *repositories.GetWorkerInjuryByIDRequest,
) (*worker.WorkerInjury, error) {
	entity := new(worker.WorkerInjury)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerInjuryScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerInjuryColumns.ID.Eq(), req.ID)
		})

	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerInjuryRelations.Worker)
	}
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerInjuryRelations.Document)
	}
	if req.IncludeEvent {
		q = q.Relation(buncolgen.WorkerInjuryRelations.SafetyEvent)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerInjury")
	}

	return entity, nil
}

func (r *repository) CreateInjury(
	ctx context.Context,
	entity *worker.WorkerInjury,
) (*worker.WorkerInjury, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			// Two cases racing for the same line on the log. The caller reads
			// the next number and inserts; on a clash the number is stale.
			return nil, errortypes.NewValidationError(
				"caseNumber",
				errortypes.ErrDuplicate,
				"That case number is already used for this year; try again",
			)
		}
		r.l.Error("failed to create injury", zap.Error(err))
		return nil, fmt.Errorf("create injury: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateInjury(
	ctx context.Context,
	entity *worker.WorkerInjury,
) (*worker.WorkerInjury, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerInjuryColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update injury", zap.Error(err))
		return nil, fmt.Errorf("update injury: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "WorkerInjury", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) DeleteInjury(
	ctx context.Context,
	req *repositories.GetWorkerInjuryByIDRequest,
) error {
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*worker.WorkerInjury)(nil)).
		Apply(func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.WorkerInjuryScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.WorkerInjuryColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete injury", zap.Error(err))
		return fmt.Errorf("delete injury: %w", err)
	}

	return dberror.CheckRowsAffected(results, "WorkerInjury", req.ID.String())
}

// NextCaseNumber is the next line on the log for the year. It reads the highest
// number rather than counting rows, so deleting a case recorded in error does
// not hand its number to the next one — an auditor reading a log with two
// case 4s has no way to tell which is which.
func (r *repository) NextCaseNumber(
	ctx context.Context,
	req *repositories.NextCaseNumberRequest,
) (int32, error) {
	cols := buncolgen.WorkerInjuryColumns

	var highest int32
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerInjury)(nil)).
		ColumnExpr("COALESCE(MAX("+cols.CaseNumber.Qualified()+"), 0)").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerInjuryScopeTenant(sq, req.TenantInfo).
				Where(cols.CaseYear.Eq(), req.CaseYear)
		}).
		Scan(ctx, &highest)
	if err != nil {
		r.l.Error("failed to read the next case number", zap.Error(err))
		return 0, fmt.Errorf("read next case number: %w", err)
	}

	return highest + 1, nil
}

func (r *repository) GetSummary(
	ctx context.Context,
	req *repositories.GetOSHASummaryRequest,
) (*worker.OSHAAnnualSummary, error) {
	entity := new(worker.OSHAAnnualSummary)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.OSHAAnnualSummaryScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.OSHAAnnualSummaryColumns.Year.Eq(), req.Year)
		}).
		Relation(buncolgen.OSHAAnnualSummaryRelations.CertifiedBy).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.l.Error("failed to read the osha summary", zap.Error(err))
		return nil, fmt.Errorf("read osha summary: %w", err)
	}

	return entity, nil
}

func (r *repository) ListSummaries(
	ctx context.Context,
	req *repositories.ListOSHASummariesRequest,
) ([]*worker.OSHAAnnualSummary, error) {
	entities := make([]*worker.OSHAAnnualSummary, 0, 8)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.OSHAAnnualSummaryScopeTenant(sq, req.TenantInfo)
		}).
		Order(buncolgen.OSHAAnnualSummaryColumns.Year.OrderDesc()).
		Limit(limitOr(req.Limit, defaultSummaryPageSize)).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list osha summaries", zap.Error(err))
		return nil, fmt.Errorf("list osha summaries: %w", err)
	}

	return entities, nil
}

func (r *repository) CreateSummary(
	ctx context.Context,
	entity *worker.OSHAAnnualSummary,
) (*worker.OSHAAnnualSummary, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"year",
				errortypes.ErrDuplicate,
				"A summary already exists for that year",
			)
		}
		r.l.Error("failed to create osha summary", zap.Error(err))
		return nil, fmt.Errorf("create osha summary: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateSummary(
	ctx context.Context,
	entity *worker.OSHAAnnualSummary,
) (*worker.OSHAAnnualSummary, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.OSHAAnnualSummaryColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update osha summary", zap.Error(err))
		return nil, fmt.Errorf("update osha summary: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		results,
		"OSHAAnnualSummary",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}
