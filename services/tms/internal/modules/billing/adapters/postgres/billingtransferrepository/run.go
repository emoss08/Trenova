package billingtransferrepository

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer/billingtransferdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingtransferports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var nonTerminalRunStatuses = []billingtransfer.RunStatus{
	billingtransfer.RunStatusQueued,
	billingtransfer.RunStatusRunning,
}

var (
	runErrors  = dberror.Mapping{Entity: "BillingTransferRun"}
	itemErrors = dberror.Mapping{Entity: "BillingTransferRunItem"}
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

func New(p Params) billingtransferports.BillingTransferRunRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.billing-transfer-run-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *billingtransfer.BillingTransferRun,
) (*billingtransfer.BillingTransferRun, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*billingtransfer.BillingTransferRun, error) {
			if err := billingtransferdb.BillingTransferRuns.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to create billing transfer run", zap.Error(err))
				return nil, runErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *billingtransfer.BillingTransferRun,
) (*billingtransfer.BillingTransferRun, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*billingtransfer.BillingTransferRun, error) {
			if err := billingtransferdb.BillingTransferRuns.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to update billing transfer run", zap.Error(err))
				return nil, runErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *billingtransferports.GetBillingTransferRunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*billingtransfer.BillingTransferRun, error) {
			entity, err := billingtransferdb.BillingTransferRuns.Get(ctx, r.db.Keel(ctx), req.RunID)
			if err != nil {
				return nil, runErrors.Map(err, req.RunID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) LastQueuedAt(
	ctx context.Context,
	req *billingtransferports.LastBillingTransferRunQueuedAtRequest,
) (int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		run := billingtransferdb.BillingTransferRuns
		queuedAt, err := keel.IntoOne[*int64](ctx, r.db.Keel(ctx), run.
			Pick(run.QueuedAt.Max()).
			Where(run.Scope.Eq(req.Scope)))
		if err != nil {
			return 0, err
		}
		if queuedAt == nil {
			return 0, nil
		}

		return *queuedAt, nil
	})
}

func (r *repository) GetActive(
	ctx context.Context,
	req *billingtransferports.GetActiveBillingTransferRunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*billingtransfer.BillingTransferRun, error) {
			run := billingtransferdb.BillingTransferRuns
			entity, err := run.Where(
				run.RequestedByID.Eq(req.RequestedByID),
				run.Status.In(nonTerminalRunStatuses...),
			).Order(run.CreatedAt.Desc()).First(ctx, r.db.Keel(ctx))
			if errors.Is(err, keel.ErrNotFound) {
				return nil, nil
			}
			if err != nil {
				r.l.Error("failed to get active billing transfer run", zap.Error(err))
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) MarkRunning(
	ctx context.Context,
	req *billingtransferports.MarkBillingTransferRunRunningRequest,
) (*billingtransfer.BillingTransferRun, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*billingtransfer.BillingTransferRun, error) {
			run := billingtransferdb.BillingTransferRuns
			return r.advanceRun(
				ctx,
				req.RunID,
				run.Status.To(billingtransfer.RunStatusRunning),
				run.TotalCount.To(int64(req.TotalCount)),
				run.UnmatchedCount.To(int64(req.UnmatchedCount)),
				run.TemporalWorkflowID.ToExpr(
					keel.Val(stringutils.NilIfEmpty(req.TemporalWorkflowID)),
				),
				run.TemporalRunID.ToExpr(keel.Val(stringutils.NilIfEmpty(req.TemporalRunID))),
				run.StartedAt.ToExpr(
					keel.Raw("COALESCE(?, ?)", run.StartedAt, timeutils.NowUnix()),
				),
			)
		},
	)
}

func (r *repository) RequestCancel(
	ctx context.Context,
	req *billingtransferports.RequestBillingTransferRunCancelRequest,
) (*billingtransfer.BillingTransferRun, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*billingtransfer.BillingTransferRun, error) {
			run := billingtransferdb.BillingTransferRuns
			return r.advanceRun(ctx, req.RunID,
				run.CancelRequestedAt.ToExpr(
					keel.Raw("COALESCE(?, ?)", run.CancelRequestedAt, timeutils.NowUnix()),
				),
				run.CancelRequestedByID.ToExpr(
					keel.Raw("COALESCE(?, ?)", run.CancelRequestedByID, req.RequestedByID.String()),
				),
			)
		},
	)
}

func (r *repository) advanceRun(
	ctx context.Context,
	runID pulid.ID,
	sets ...keel.Assignment,
) (*billingtransfer.BillingTransferRun, error) {
	run := billingtransferdb.BillingTransferRuns
	rows, err := run.Where(run.ID.Eq(runID), run.Status.In(nonTerminalRunStatuses...)).
		UpdateReturning(ctx, r.db.Keel(ctx), sets...)
	if err != nil {
		r.l.Error("failed to advance billing transfer run", zap.Error(err))
		return nil, runErrors.Map(err, runID.String())
	}
	if err = dberror.CheckAffected(
		int64(len(rows)),
		"BillingTransferRun",
		runID.String(),
	); err != nil {
		return nil, err
	}

	return rows[0], nil
}

func (r *repository) ListStale(
	ctx context.Context,
	req *billingtransferports.ListStaleBillingTransferRunsRequest,
) ([]*billingtransfer.BillingTransferRun, error) {
	ctx = dbscope.WithSystem(ctx, "list stalled billing transfer runs across every organization")
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingtransfer.BillingTransferRun, error) {
			statuses := req.Statuses
			if len(statuses) == 0 {
				statuses = nonTerminalRunStatuses
			}

			run := billingtransferdb.BillingTransferRuns
			q := run.AllTenants().
				Where(run.Status.In(statuses...), run.UpdatedAt.Lt(req.UpdatedBeforeUnix)).
				Order(run.UpdatedAt.Asc())

			var (
				entities []*billingtransfer.BillingTransferRun
				err      error
			)
			if req.Limit > 0 {
				entities, err = q.Limit(req.Limit).List(ctx, r.db.Keel(ctx))
			} else {
				entities, err = q.All(ctx, r.db.Keel(ctx))
			}
			if err != nil {
				r.l.Error("failed to list stale billing transfer runs", zap.Error(err))
				return nil, err
			}
			if entities == nil {
				return []*billingtransfer.BillingTransferRun{}, nil
			}

			return entities, nil
		},
	)
}
