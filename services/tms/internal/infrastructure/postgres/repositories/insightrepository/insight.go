// Package insightrepository stores operational findings and reads the
// aggregates the detectors measure them from.
package insightrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// defaultActiveLimit bounds an unbounded read. A home widget asks for a handful;
// this only catches a caller that forgot to say.
const defaultActiveLimit = 50

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.InsightRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.insight-repository"),
	}
}

// severityOrder sorts by urgency rather than alphabetically, which would put
// Critical after Info and bury the thing worth reading.
const severityOrder = `CASE ins.severity
	WHEN 'Critical' THEN 3
	WHEN 'Warning' THEN 2
	WHEN 'Info' THEN 1
	ELSE 0 END DESC`

func (r *repository) ListActive(
	ctx context.Context,
	req repositories.ListActiveInsightsRequest,
) ([]*insight.Insight, error) {
	log := r.l.With(zap.String("operation", "ListActive"))

	limit := req.Limit
	if limit <= 0 || limit > defaultActiveLimit {
		limit = defaultActiveLimit
	}

	cols := buncolgen.InsightColumns
	entities := make([]*insight.Insight, 0, limit)

	query := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(buncolgen.InsightApplyTenant(req.TenantInfo)).
		Where(cols.Status.Eq(), insight.StatusActive)

	if len(req.Categories) > 0 {
		query = query.Where(cols.Category.In(), bun.In(req.Categories))
	}

	if err := query.
		OrderExpr(severityOrder).
		Order(cols.DetectedAt.OrderDesc()).
		Limit(limit).
		Scan(ctx); err != nil {
		log.Error("failed to list active insights", zap.Error(err))

		return nil, err
	}

	return entities, nil
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListInsightRequest,
) (*pagination.ListResult[*insight.Insight], error) {
	log := r.l.With(zap.String("operation", "List"))

	cols := buncolgen.InsightColumns
	entities := make([]*insight.Insight, 0, req.Filter.Pagination.SafeLimit())

	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = querybuilder.ApplyFilters(
				sq,
				buncolgen.InsightTable.Alias,
				req.Filter,
				(*insight.Insight)(nil),
			)

			return sq.Apply(buncolgen.InsightApplyTenant(req.Filter.TenantInfo)).
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset()).
				Order(cols.DetectedAt.OrderDesc())
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count insights", zap.Error(err))

		return nil, err
	}

	return &pagination.ListResult[*insight.Insight]{Items: entities, Total: total}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetInsightByIDRequest,
) (*insight.Insight, error) {
	log := r.l.With(zap.String("operation", "GetByID"), zap.String("id", req.ID.String()))

	cols := buncolgen.InsightColumns
	entity := new(insight.Insight)

	if err := r.db.DB().
		NewSelect().
		Model(entity).
		Apply(buncolgen.InsightApplyTenant(req.TenantInfo)).
		Where(cols.ID.Eq(), req.ID).
		Scan(ctx); err != nil {
		log.Error("failed to get insight", zap.Error(err))

		return nil, dberror.HandleNotFoundError(err, "Insight")
	}

	return entity, nil
}

// Dismiss records a person's judgement that a finding is not worth acting on.
//
// It is restricted to an active insight: dismissing one that a refresh has
// already resolved or superseded would resurrect a row nobody is looking at and
// suppress a finding that is no longer being made.
func (r *repository) Dismiss(
	ctx context.Context,
	req repositories.DismissInsightRequest,
) (*insight.Insight, error) {
	log := r.l.With(zap.String("operation", "Dismiss"), zap.String("id", req.ID.String()))

	cols := buncolgen.InsightColumns
	entity := new(insight.Insight)
	now := timeutils.NowUnix()

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.InsightScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID).
				Where(cols.Status.Eq(), insight.StatusActive)
		}).
		Set(cols.Status.Set(), insight.StatusDismissed).
		Set(cols.DismissedAt.Set(), now).
		Set(cols.DismissedByID.Set(), req.UserID).
		Set(cols.DismissReason.Set(), req.Reason).
		Set(cols.UpdatedAt.Set(), now).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to dismiss insight", zap.Error(err))

		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Insight", req.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

// ReplaceDetectorFindings applies one detector's whole result in a transaction.
//
// The three steps only make sense together. Superseding the previous run before
// inserting the new one keeps the unique index on active dedupe keys satisfied;
// resolving what the detector no longer finds is what makes a fixed problem
// disappear from the home screen rather than sitting there indefinitely. Doing
// them in separate transactions would leave a reader looking at a moment where a
// finding had been superseded and not yet replaced.
func (r *repository) ReplaceDetectorFindings(
	ctx context.Context,
	req repositories.ReplaceDetectorFindingsRequest,
) (repositories.ReplaceDetectorFindingsResult, error) {
	log := r.l.With(
		zap.String("operation", "ReplaceDetectorFindings"),
		zap.String("detector", req.DetectorKey),
	)

	var result repositories.ReplaceDetectorFindingsResult

	err := r.db.DB().RunInTx(ctx, nil, func(txCtx context.Context, tx bun.Tx) error {
		suppressed, err := r.suppressedKeys(txCtx, tx, req)
		if err != nil {
			return err
		}

		keep := make([]*insight.Insight, 0, len(req.Insights))
		for _, entity := range req.Insights {
			if _, isSuppressed := suppressed[entity.DedupeKey]; isSuppressed {
				result.Suppressed++
				continue
			}
			keep = append(keep, entity)
		}

		counts, err := r.applyReplacement(txCtx, tx, req, keep)
		if err != nil {
			return err
		}

		result.Superseded = counts.Superseded
		result.Resolved = counts.Resolved
		result.Created = counts.Created

		return nil
	})
	if err != nil {
		log.Error("failed to replace detector findings", zap.Error(err))

		return repositories.ReplaceDetectorFindingsResult{}, err
	}

	return result, nil
}

// suppressedKeys reads the dedupe keys a person dismissed recently enough that
// the finding should not come back.
func (r *repository) suppressedKeys(
	ctx context.Context,
	tx bun.Tx,
	req repositories.ReplaceDetectorFindingsRequest,
) (map[string]struct{}, error) {
	cols := buncolgen.InsightColumns
	keys := make([]string, 0)

	if err := tx.NewSelect().
		Model((*insight.Insight)(nil)).
		Column(cols.DedupeKey.String()).
		Apply(buncolgen.InsightApplyTenant(req.TenantInfo)).
		Where(cols.DetectorKey.Eq(), req.DetectorKey).
		Where(cols.Status.Eq(), insight.StatusDismissed).
		Where(cols.DismissedAt.Gte(), req.SuppressedBefore).
		Scan(ctx, &keys); err != nil {
		return nil, fmt.Errorf("read suppressed findings: %w", err)
	}

	suppressed := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		suppressed[key] = struct{}{}
	}

	return suppressed, nil
}

func (r *repository) applyReplacement(
	ctx context.Context,
	tx bun.Tx,
	req repositories.ReplaceDetectorFindingsRequest,
	keep []*insight.Insight,
) (repositories.ReplaceDetectorFindingsResult, error) {
	var result repositories.ReplaceDetectorFindingsResult

	current := make([]string, 0, len(keep))
	for _, entity := range keep {
		current = append(current, entity.DedupeKey)
	}

	superseded, err := r.closeActive(ctx, tx, closeParams{
		req:    req,
		status: insight.StatusSuperseded,
		keys:   current,
		within: true,
	})
	if err != nil {
		return result, err
	}
	result.Superseded = superseded

	resolved, err := r.closeActive(ctx, tx, closeParams{
		req:    req,
		status: insight.StatusResolved,
		keys:   current,
		within: false,
	})
	if err != nil {
		return result, err
	}
	result.Resolved = resolved

	if len(keep) == 0 {
		return result, nil
	}

	if _, err = tx.NewInsert().Model(&keep).Exec(ctx); err != nil {
		return result, fmt.Errorf("insert findings: %w", err)
	}
	result.Created = len(keep)

	return result, nil
}

type closeParams struct {
	req    repositories.ReplaceDetectorFindingsRequest
	status insight.Status
	keys   []string
	// within selects whether to close the rows whose dedupe key is in keys
	// (superseded by a fresher run) or the rows whose key is not (the condition
	// has stopped being true).
	within bool
}

func (r *repository) closeActive(
	ctx context.Context,
	tx bun.Tx,
	params closeParams,
) (int, error) {
	cols := buncolgen.InsightColumns
	now := timeutils.NowUnix()

	query := tx.NewUpdate().
		Model((*insight.Insight)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.InsightScopeTenantUpdate(uq, params.req.TenantInfo).
				Where(cols.DetectorKey.Eq(), params.req.DetectorKey).
				Where(cols.Status.Eq(), insight.StatusActive)
		}).
		Set(cols.Status.Set(), params.status).
		Set(cols.UpdatedAt.Set(), now)

	switch {
	case len(params.keys) > 0 && params.within:
		query = query.Where(cols.DedupeKey.In(), bun.In(params.keys))
	case len(params.keys) > 0:
		query = query.Where(cols.DedupeKey.NotIn(), bun.In(params.keys))
	case params.within:
		// Nothing was found this run, so nothing is being superseded. Without
		// this guard the empty key list would match every active row and mark it
		// superseded as well as resolved.
		return 0, nil
	}

	results, err := query.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("close active findings as %s: %w", params.status, err)
	}

	affected, err := results.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count closed findings: %w", err)
	}

	return int(affected), nil
}
