package workerleaverepository

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
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultCasePageSize  = 200
	defaultEntryPageSize = 1000
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

func New(p Params) repositories.WorkerLeaveRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-leave-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

// GetControl reads the organisation's leave settings, creating the row with the
// statutory defaults if it is missing. It never returns nil: an absent control
// would read as an entitlement of zero, which would report every worker as
// having exhausted their leave.
func (r *repository) GetControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*worker.LeaveControl, error) {
	entity := new(worker.LeaveControl)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.LeaveControlScopeTenant(sq, tenantInfo)
		}).
		Limit(1).
		Scan(ctx)
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		r.l.Error("failed to read leave control", zap.Error(err))
		return nil, fmt.Errorf("read leave control: %w", err)
	}

	created := worker.DefaultLeaveControl()
	created.OrganizationID = tenantInfo.OrgID
	created.BusinessUnitID = tenantInfo.BuID
	if _, iErr := r.db.DBForContext(ctx).
		NewInsert().
		Model(created).
		On("CONFLICT DO NOTHING").
		Exec(ctx); iErr != nil {
		r.l.Error("failed to seed leave control", zap.Error(iErr))
		return nil, fmt.Errorf("seed leave control: %w", iErr)
	}

	return created, nil
}

func (r *repository) UpdateControl(
	ctx context.Context,
	entity *worker.LeaveControl,
) (*worker.LeaveControl, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.LeaveControlColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update leave control", zap.Error(err))
		return nil, fmt.Errorf("update leave control: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "LeaveControl", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

// caseFilter is the one place the case predicates live, so a count and a list
// can never disagree about what they are counting.
func caseFilter(
	req *repositories.ListLeaveCasesRequest,
) func(*bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.WorkerLeaveCaseColumns
	return func(sq *bun.SelectQuery) *bun.SelectQuery {
		sq = buncolgen.WorkerLeaveCaseScopeTenant(sq, req.TenantInfo)
		if !req.WorkerID.IsNil() {
			sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
		}
		if req.OpenOnly {
			sq = sq.Where(cols.Status.In(), bun.In([]worker.LeaveCaseStatus{
				worker.LeaveCasePending,
				worker.LeaveCaseApproved,
			}))
		}
		if req.CertificationOutstandingOnly {
			sq = sq.Where(
				cols.CertificationStatus.In(),
				bun.In([]worker.LeaveCertificationStatus{
					worker.CertificationRequested,
					worker.CertificationInsufficient,
					worker.CertificationOverdue,
				}),
			)
		}
		return sq
	}
}

func (r *repository) ListCases(
	ctx context.Context,
	req *repositories.ListLeaveCasesRequest,
) ([]*worker.WorkerLeaveCase, error) {
	cols := buncolgen.WorkerLeaveCaseColumns
	entities := make([]*worker.WorkerLeaveCase, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", caseFilter(req)).
		Order(cols.StartsAt.OrderDesc()).
		Limit(limitOr(req.Limit, defaultCasePageSize))

	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerLeaveCaseRelations.Worker)
	}
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerLeaveCaseRelations.Document)
	}
	if req.IncludeEntries {
		q = q.Relation(buncolgen.WorkerLeaveCaseRelations.Entries)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list leave cases", zap.Error(err))
		return nil, fmt.Errorf("list leave cases: %w", err)
	}

	return entities, nil
}

func (r *repository) CountCases(
	ctx context.Context,
	req *repositories.ListLeaveCasesRequest,
) (int, error) {
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerLeaveCase)(nil)).
		WhereGroup(" AND ", caseFilter(req)).
		Count(ctx)
	if err != nil {
		r.l.Error("failed to count leave cases", zap.Error(err))
		return 0, fmt.Errorf("count leave cases: %w", err)
	}

	return total, nil
}

func (r *repository) GetCaseByID(
	ctx context.Context,
	req *repositories.GetLeaveCaseByIDRequest,
) (*worker.WorkerLeaveCase, error) {
	entity := new(worker.WorkerLeaveCase)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerLeaveCaseScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerLeaveCaseColumns.ID.Eq(), req.ID)
		})
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerLeaveCaseRelations.Document)
	}
	if req.IncludeEntries {
		q = q.Relation(buncolgen.WorkerLeaveCaseRelations.Entries)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerLeaveCase")
	}

	return entity, nil
}

func (r *repository) CreateCase(
	ctx context.Context,
	entity *worker.WorkerLeaveCase,
) (*worker.WorkerLeaveCase, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create leave case", zap.Error(err))
		return nil, fmt.Errorf("create leave case: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateCase(
	ctx context.Context,
	entity *worker.WorkerLeaveCase,
) (*worker.WorkerLeaveCase, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerLeaveCaseColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update leave case", zap.Error(err))
		return nil, fmt.Errorf("update leave case: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "WorkerLeaveCase", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListEntries(
	ctx context.Context,
	req *repositories.ListLeaveEntriesRequest,
) ([]*worker.WorkerLeaveEntry, error) {
	cols := buncolgen.WorkerLeaveEntryColumns
	entities := make([]*worker.WorkerLeaveEntry, 0, 32)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerLeaveEntryScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if !req.LeaveCaseID.IsNil() {
				sq = sq.Where(cols.LeaveCaseID.Eq(), req.LeaveCaseID)
			}
			if req.Since > 0 {
				sq = sq.Where(cols.UsedOn.Gte(), req.Since)
			}
			return sq
		}).
		Order(cols.UsedOn.OrderDesc()).
		Limit(limitOr(req.Limit, defaultEntryPageSize)).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list leave entries", zap.Error(err))
		return nil, fmt.Errorf("list leave entries: %w", err)
	}

	return entities, nil
}

func (r *repository) ListEntriesByCaseIDs(
	ctx context.Context,
	req *repositories.ListLeaveEntriesByCaseIDsRequest,
) (map[pulid.ID][]*worker.WorkerLeaveEntry, error) {
	cols := buncolgen.WorkerLeaveEntryColumns
	entities := make([]*worker.WorkerLeaveEntry, 0, len(req.LeaveCaseIDs)*32)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerLeaveEntryScopeTenant(sq, req.TenantInfo).
				Where(cols.LeaveCaseID.In(), bun.List(req.LeaveCaseIDs))
		}).
		Order(cols.UsedOn.OrderDesc()).
		Limit(len(req.LeaveCaseIDs) * defaultEntryPageSize).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list leave entries by cases", zap.Error(err))
		return nil, fmt.Errorf("list leave entries by cases: %w", err)
	}

	return sliceutils.GroupBy(entities, func(e *worker.WorkerLeaveEntry) pulid.ID {
		return e.LeaveCaseID
	}), nil
}

func (r *repository) GetEntryByID(
	ctx context.Context,
	req *repositories.GetLeaveEntryByIDRequest,
) (*worker.WorkerLeaveEntry, error) {
	entity := new(worker.WorkerLeaveEntry)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerLeaveEntryScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerLeaveEntryColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerLeaveEntry")
	}

	return entity, nil
}

func (r *repository) CreateEntry(
	ctx context.Context,
	entity *worker.WorkerLeaveEntry,
) (*worker.WorkerLeaveEntry, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"usedOn",
				errortypes.ErrDuplicate,
				"That day is already recorded against this case; edit it instead",
			)
		}
		r.l.Error("failed to create leave entry", zap.Error(err))
		return nil, fmt.Errorf("create leave entry: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateEntry(
	ctx context.Context,
	entity *worker.WorkerLeaveEntry,
) (*worker.WorkerLeaveEntry, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerLeaveEntryColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update leave entry", zap.Error(err))
		return nil, fmt.Errorf("update leave entry: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		results,
		"WorkerLeaveEntry",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) DeleteEntry(
	ctx context.Context,
	req *repositories.GetLeaveEntryByIDRequest,
) error {
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*worker.WorkerLeaveEntry)(nil)).
		Apply(func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.WorkerLeaveEntryScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.WorkerLeaveEntryColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete leave entry", zap.Error(err))
		return fmt.Errorf("delete leave entry: %w", err)
	}

	return dberror.CheckRowsAffected(results, "WorkerLeaveEntry", req.ID.String())
}
