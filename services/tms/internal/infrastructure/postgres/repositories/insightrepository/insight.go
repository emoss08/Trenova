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
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// defaultActiveLimit bounds an unbounded read. A home widget asks for a
	// handful; this only catches a caller that forgot to say.
	defaultActiveLimit = 50
	defaultPageLimit   = 25
	maxPageLimit       = 100
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

	if len(req.AllowedDetectorKeys) == 0 {
		// No detector is visible to this reader, so there is nothing to ask the
		// database for. Returning early also keeps an empty IN () out of the query,
		// which Postgres treats as matching nothing but Bun renders awkwardly.
		return entities, nil
	}

	query := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(buncolgen.InsightApplyTenant(req.TenantInfo)).
		Where(cols.Status.Eq(), insight.StatusActive).
		Where(cols.DetectorKey.In(), bun.In([]string(req.AllowedDetectorKeys)))

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

// List browses the whole history a page at a time.
//
// Every filter is applied in the query, including the detector permission, so
// the total is a count of what this reader can actually receive. A page that
// filtered afterwards would report a total nobody can reach and hand back short
// pages with no explanation.
func (r *repository) List(
	ctx context.Context,
	req repositories.ListInsightsRequest,
) (*pagination.ListResult[*insight.Insight], error) {
	log := r.l.With(zap.String("operation", "List"))

	cols := buncolgen.InsightColumns
	limit, offset := pageBounds(req.Limit, req.Offset)
	entities := make([]*insight.Insight, 0, limit)

	if len(req.AllowedDetectorKeys) == 0 {
		return &pagination.ListResult[*insight.Insight]{Items: entities, Total: 0}, nil
	}

	query := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(buncolgen.InsightApplyTenant(req.TenantInfo)).
		Where(cols.DetectorKey.In(), bun.In([]string(req.AllowedDetectorKeys)))

	if len(req.Statuses) > 0 {
		query = query.Where(cols.Status.In(), bun.In(req.Statuses))
	} else {
		// Someone opening the page is asking what needs attention now. Seeing what
		// was dismissed or has since resolved is a deliberate act, not the default.
		query = query.Where(cols.Status.Eq(), insight.StatusActive)
	}

	if len(req.Categories) > 0 {
		query = query.Where(cols.Category.In(), bun.In(req.Categories))
	}

	if len(req.Severities) > 0 {
		query = query.Where(cols.Severity.In(), bun.In(req.Severities))
	}

	total, err := query.
		OrderExpr(severityOrder).
		Order(cols.DetectedAt.OrderDesc()).
		Limit(limit).
		Offset(offset).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count insights", zap.Error(err))

		return nil, err
	}

	return &pagination.ListResult[*insight.Insight]{Items: entities, Total: total}, nil
}

// pageBounds keeps a caller that asked for nothing, or for everything, inside
// what one page is meant to be.
func pageBounds(limit, offset int) (int, int) {
	if limit <= 0 || limit > maxPageLimit {
		limit = defaultPageLimit
	}

	if offset < 0 {
		offset = 0
	}

	return limit, offset
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

// Restore returns a dismissed finding to active.
//
// Only a dismissed one: a resolved insight describes a condition a later refresh
// could no longer find, and putting that back would be asserting something the
// data does not support. The dismissal columns are cleared rather than kept, so
// the next refresh stops suppressing the finding and the card behaves as though
// it had never been waved away.
func (r *repository) Restore(
	ctx context.Context,
	req repositories.RestoreInsightRequest,
) (*insight.Insight, error) {
	log := r.l.With(zap.String("operation", "Restore"), zap.String("id", req.ID.String()))

	cols := buncolgen.InsightColumns
	entity := new(insight.Insight)

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.InsightScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID).
				Where(cols.Status.Eq(), insight.StatusDismissed)
		}).
		Set(cols.Status.Set(), insight.StatusActive).
		Set(cols.DismissedAt.SetNull()).
		Set(cols.DismissedByID.SetNull()).
		Set(cols.DismissReason.SetNull()).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to restore insight", zap.Error(err))

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
