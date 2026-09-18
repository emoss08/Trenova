package billingtransferrepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// nonTerminalRunStatuses is the set a run can still be worked on in. Every
// mutation that advances a run guards on it, so a canceled or finished run is
// never reopened by a late-arriving activity.
var nonTerminalRunStatuses = []billingtransfer.RunStatus{
	billingtransfer.RunStatusQueued,
	billingtransfer.RunStatusRunning,
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.BillingTransferRunRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.billing-transfer-run-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *billingtransfer.BillingTransferRun,
) (*billingtransfer.BillingTransferRun, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		r.l.Error("failed to create billing transfer run", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *billingtransfer.BillingTransferRun,
) (*billingtransfer.BillingTransferRun, error) {
	log := r.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))

	ov := entity.Version
	entity.Version++

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.BillingTransferRunColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update billing transfer run", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "BillingTransferRun", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetBillingTransferRunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	entity := new(billingtransfer.BillingTransferRun)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.BillingTransferRunScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.BillingTransferRunColumns.ID.Eq(), req.RunID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "BillingTransferRun")
	}

	return entity, nil
}

// GetActive returns the one run this user has not finished. Having none is the
// ordinary case, so it answers with nothing rather than a not-found error; the
// caller's dialog simply opens on the picker instead of reattaching.
func (r *repository) GetActive(
	ctx context.Context,
	req *repositories.GetActiveBillingTransferRunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	cols := buncolgen.BillingTransferRunColumns

	entity := new(billingtransfer.BillingTransferRun)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.BillingTransferRunApplyTenant(req.TenantInfo)).
		Where(cols.RequestedByID.Eq(), req.RequestedByID).
		Where(cols.Status.In(), bun.List(nonTerminalRunStatuses)).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // no run in flight is an answer, not a failure
		}
		r.l.Error("failed to get active billing transfer run", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// MarkRunning stamps the Temporal identity on the run and opens it for work.
//
// It is written to be repeatable, because the activity that calls it can be
// retried: StartedAt is only taken the first time, and the status guard means a
// run somebody canceled in the meantime is not dragged back into Running.
func (r *repository) MarkRunning(
	ctx context.Context,
	req *repositories.MarkBillingTransferRunRunningRequest,
) (*billingtransfer.BillingTransferRun, error) {
	cols := buncolgen.BillingTransferRunColumns
	now := timeutils.NowUnix()

	entity := new(billingtransfer.BillingTransferRun)
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set(cols.Status.Set(), billingtransfer.RunStatusRunning).
		Set(cols.TotalCount.Set(), req.TotalCount).
		Set(cols.UnmatchedCount.Set(), req.UnmatchedCount).
		Set(cols.TemporalWorkflowID.Set(), req.TemporalWorkflowID).
		Set(cols.TemporalRunID.Set(), req.TemporalRunID).
		Set(cols.StartedAt.SetExpr("COALESCE({}, ?)"), now).
		Set(cols.UpdatedAt.Set(), now).
		Set(cols.Version.Inc(1)).
		Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.BillingTransferRunScopeTenantUpdate(uq, req.TenantInfo)
		}).
		Where(cols.ID.Eq(), req.RunID).
		Where(cols.Status.In(), bun.List(nonTerminalRunStatuses)).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to mark billing transfer run running", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "BillingTransferRun", req.RunID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

// RequestCancel records the ask and nothing more.
//
// It deliberately leaves the run non-terminal: the workflow stops between
// batches and finalizes itself, so the batch already in flight is never
// abandoned with its outcomes unwritten. COALESCE keeps the first asker.
func (r *repository) RequestCancel(
	ctx context.Context,
	req *repositories.RequestBillingTransferRunCancelRequest,
) (*billingtransfer.BillingTransferRun, error) {
	cols := buncolgen.BillingTransferRunColumns
	now := timeutils.NowUnix()

	entity := new(billingtransfer.BillingTransferRun)
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set(cols.CancelRequestedAt.SetExpr("COALESCE({}, ?)"), now).
		Set(cols.CancelRequestedByID.SetExpr("COALESCE({}, ?)"), req.RequestedByID).
		Set(cols.UpdatedAt.Set(), now).
		Set(cols.Version.Inc(1)).
		Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.BillingTransferRunScopeTenantUpdate(uq, req.TenantInfo)
		}).
		Where(cols.ID.Eq(), req.RunID).
		Where(cols.Status.In(), bun.List(nonTerminalRunStatuses)).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to request billing transfer run cancel", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "BillingTransferRun", req.RunID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListStale(
	ctx context.Context,
	req *repositories.ListStaleBillingTransferRunsRequest,
) ([]*billingtransfer.BillingTransferRun, error) {
	cols := buncolgen.BillingTransferRunColumns

	statuses := req.Statuses
	if len(statuses) == 0 {
		statuses = nonTerminalRunStatuses
	}

	entities := make([]*billingtransfer.BillingTransferRun, 0)
	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Where(cols.Status.In(), bun.List(statuses)).
		Where(cols.UpdatedAt.Lt(), req.UpdatedBeforeUnix).
		Order(cols.UpdatedAt.OrderAsc())

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list stale billing transfer runs", zap.Error(err))
		return nil, err
	}

	return entities, nil
}
