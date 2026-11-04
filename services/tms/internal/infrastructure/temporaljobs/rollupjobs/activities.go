package rollupjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel/analytics/rollup"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// batchSize is how many dirty marks one transaction claims; the distinct tenant days among
	// them are recomputed together.
	batchSize = 500
	// maxBatchesPerRun bounds one run, so a backlog (a migration marking every tenant) drains
	// over several runs and each run's activity stays short.
	maxBatchesPerRun = 200
	// statementTimeout bounds one recompute: a whole tenant rebuilt after a migration or a time
	// zone change reads every settled row the tenant has.
	statementTimeout = 5 * time.Minute
	heartbeatEvery   = 10 * time.Second
	systemReason     = "refresh every tenant's rollup days that writes marked dirty"
)

type ActivitiesParams struct {
	fx.In

	DB      *postgres.Connection
	Rollups Rollups
	Metrics *metrics.Registry
	Logger  *zap.Logger
}

type Activities struct {
	db      *postgres.Connection
	rollups []*rollup.Rollup
	metrics *metrics.Registry
	l       *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		db:      p.DB,
		rollups: p.Rollups,
		metrics: p.Metrics,
		l:       p.Logger.Named("temporal.rollup-refresh"),
	}
}

// RefreshRollupsActivity recomputes the dirty days of every rollup, one transaction per batch,
// until none is left or the run's bound is reached. Each batch commits on its own, so a retry
// after a failure redoes only what was still marked.
func (a *Activities) RefreshRollupsActivity(ctx context.Context) (*RefreshResult, error) {
	if len(a.rollups) == 0 {
		return &RefreshResult{}, nil
	}
	stop := temporaltype.HeartbeatEvery(ctx, heartbeatEvery)
	defer stop()

	ctx = dbscope.WithSystem(ctx, systemReason)
	db := a.db.Keel(ctx)
	start := time.Now()
	result := &RefreshResult{}
	options := rollup.RefreshOptions{
		BatchSize:        batchSize,
		MaxBatches:       1,
		StatementTimeout: statementTimeout,
	}

	for result.Batches < maxBatchesPerRun {
		report, err := rollup.RefreshWith(ctx, db, options, a.rollups...)
		a.record(&report)
		if err != nil {
			return nil, fmt.Errorf("refresh rollups: %w", err)
		}
		if report.Slices+report.Tenants == 0 {
			break
		}
		result.Batches++
		result.Slices += report.Slices
		result.Tenants += report.Tenants
		result.Rows += report.Rows
		if err = ctx.Err(); err != nil {
			return nil, err
		}
	}
	a.metrics.Database.ObserveRollupRefresh(time.Since(start))

	pending, err := rollup.Pending(ctx, db, a.rollups...)
	if err != nil {
		return nil, fmt.Errorf("count pending rollup marks: %w", err)
	}
	for name, marks := range pending {
		a.metrics.Database.SetRollupPending(name, marks)
		result.Pending += marks
	}

	if result.Batches > 0 {
		a.l.Debug("rollups refreshed",
			zap.Int("batches", result.Batches),
			zap.Int("slices", result.Slices),
			zap.Int("tenants", result.Tenants),
			zap.Int64("rows", result.Rows),
			zap.Int64("pending", result.Pending),
			zap.Duration("elapsed", time.Since(start)),
		)
	}

	return result, nil
}

func (a *Activities) record(report *rollup.Report) {
	for name, done := range report.Rollups {
		a.metrics.Database.RecordRollupRefresh(name, done.Slices, done.Tenants, done.Rows)
	}
}
