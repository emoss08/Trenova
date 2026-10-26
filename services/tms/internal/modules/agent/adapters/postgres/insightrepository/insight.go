package insightrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight/insightdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultActiveLimit = 50
	defaultPageLimit   = 25
	maxPageLimit       = 100
	maxHistoryLimit    = 12
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

func New(p Params) insightports.InsightRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.insight-repository"),
	}
}

func severityRank() keel.Expr {
	severity := insightdb.Insights.Severity
	rank := keel.Int64

	return keel.DynCase().
		When(severity.Eq(insight.SeverityCritical), rank(3)).
		When(severity.Eq(insight.SeverityWarning), rank(2)).
		When(severity.Eq(insight.SeverityInfo), rank(1)).
		Else(rank(0))
}

func urgentFirst() []keel.Order {
	return []keel.Order{keel.Desc(severityRank()), insightdb.Insights.DetectedAt.Desc()}
}

func (r *repository) ListActive(
	ctx context.Context,
	req insightports.ListActiveInsightsRequest,
) ([]*insight.Insight, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*insight.Insight, error) {
		if len(req.AllowedDetectorKeys) == 0 {
			return []*insight.Insight{}, nil
		}

		entities, err := activeQuery(&req).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list active insights",
				zap.String("operation", "ListActive"),
				zap.Error(err))

			return nil, err
		}

		return entities, nil
	})
}

func activeQuery(req *insightports.ListActiveInsightsRequest) *keel.Query[*insight.Insight] {
	limit := req.Limit
	if limit <= 0 || limit > defaultActiveLimit {
		limit = defaultActiveLimit
	}

	in := insightdb.Insights
	return in.Where(
		in.Status.Eq(insight.StatusActive),
		in.DetectorKey.In(req.AllowedDetectorKeys...),
		keel.When(len(req.Categories) > 0, in.Category.In(req.Categories...)),
	).
		Order(urgentFirst()...).
		Limit(limit)
}

func (r *repository) List(
	ctx context.Context,
	req insightports.ListInsightsQuery,
) (*pagination.ListResult[*insight.Insight], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*insight.Insight], error) {
			if len(req.AllowedDetectorKeys) == 0 {
				return &pagination.ListResult[*insight.Insight]{
					Items:      []*insight.Insight{},
					TotalExact: true,
				}, nil
			}

			limit, offset := pageBounds(req.Limit, req.Offset)
			filtered := listFilter(&req)

			var (
				entities []*insight.Insight
				total    keel.RowCount
			)
			if err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.ListInto(
					filtered.Clone().Order(urgentFirst()...).Limit(limit).Offset(offset),
					&entities,
				),
				keel.CountEstimateInto(filtered, pagination.ExactTotalUpTo, &total),
			); err != nil {
				r.l.Error("failed to scan and count insights",
					zap.String("operation", "List"),
					zap.Error(err))

				return nil, err
			}
			if entities == nil {
				entities = []*insight.Insight{}
			}

			return listquery.Result(entities, total), nil
		},
	)
}

func listFilter(req *insightports.ListInsightsQuery) *keel.Query[*insight.Insight] {
	in := insightdb.Insights

	status := in.Status.Eq(insight.StatusActive)
	if len(req.Statuses) > 0 {
		status = in.Status.In(req.Statuses...)
	}

	return in.Where(
		in.DetectorKey.In(req.AllowedDetectorKeys...),
		status,
		keel.When(len(req.Categories) > 0, in.Category.In(req.Categories...)),
		keel.When(len(req.Severities) > 0, in.Severity.In(req.Severities...)),
	)
}

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
	req insightports.GetInsightByIDRequest,
) (*insight.Insight, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*insight.Insight, error) {
		entity, err := insightdb.Insights.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			r.l.Error("failed to get insight",
				zap.String("operation", "GetByID"),
				zap.String("id", req.ID.String()),
				zap.Error(err))

			return nil, dberror.HandleNotFoundError(err, "Insight")
		}

		return entity, nil
	})
}

func (r *repository) ListHistory(
	ctx context.Context,
	req insightports.ListInsightHistoryRequest,
) ([]*insight.Insight, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*insight.Insight, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxHistoryLimit {
			limit = maxHistoryLimit
		}

		in := insightdb.Insights
		entities, err := in.Where(
			in.DedupeKey.Eq(req.DedupeKey),
			keel.When(!req.ExcludeID.IsNil(), in.ID.Ne(req.ExcludeID)),
		).
			Order(in.DetectedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list insight history",
				zap.String("operation", "ListHistory"),
				zap.Error(err))

			return nil, err
		}

		return entities, nil
	})
}

func keepInsightVersion() keel.Assignment {
	in := insightdb.Insights
	return in.Version.ToExpr(in.Version)
}

func (r *repository) Dismiss(
	ctx context.Context,
	req insightports.DismissInsightQuery,
) (*insight.Insight, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*insight.Insight, error) {
		in := insightdb.Insights
		rows, err := in.Where(in.ID.Eq(req.ID), in.Status.Eq(insight.StatusActive)).
			UpdateReturning(ctx, r.db.Keel(ctx),
				in.Status.To(insight.StatusDismissed),
				in.DismissedAt.To(timeutils.NowUnix()),
				in.DismissedByID.To(req.UserID),
				in.DismissReason.To(req.Reason),
				keepInsightVersion(),
			)
		if err != nil {
			r.l.Error("failed to dismiss insight",
				zap.String("operation", "Dismiss"),
				zap.String("id", req.ID.String()),
				zap.Error(err))

			return nil, err
		}

		if err = dberror.CheckAffected(int64(len(rows)), "Insight", req.ID.String()); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}

func (r *repository) Restore(
	ctx context.Context,
	req insightports.RestoreInsightQuery,
) (*insight.Insight, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*insight.Insight, error) {
		in := insightdb.Insights
		rows, err := in.Where(in.ID.Eq(req.ID), in.Status.Eq(insight.StatusDismissed)).
			UpdateReturning(ctx, r.db.Keel(ctx),
				in.Status.To(insight.StatusActive),
				in.DismissedAt.ToNull(),
				in.DismissedByID.ToNull(),
				in.DismissReason.ToNull(),
				keepInsightVersion(),
			)
		if err != nil {
			r.l.Error("failed to restore insight",
				zap.String("operation", "Restore"),
				zap.String("id", req.ID.String()),
				zap.Error(err))

			return nil, err
		}

		if err = dberror.CheckAffected(int64(len(rows)), "Insight", req.ID.String()); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}

func (r *repository) ReplaceDetectorFindings(
	ctx context.Context,
	req insightports.ReplaceDetectorFindingsRequest,
) (insightports.ReplaceDetectorFindingsResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (insightports.ReplaceDetectorFindingsResult, error) {
			result, err := r.replace(ctx, &req)
			if err != nil {
				r.l.Error("failed to replace detector findings",
					zap.String("operation", "ReplaceDetectorFindings"),
					zap.String("detector", req.DetectorKey),
					zap.Error(err))

				return insightports.ReplaceDetectorFindingsResult{}, err
			}

			return result, nil
		},
	)
}

func (r *repository) replace(
	ctx context.Context,
	req *insightports.ReplaceDetectorFindingsRequest,
) (insightports.ReplaceDetectorFindingsResult, error) {
	var result insightports.ReplaceDetectorFindingsResult
	db := r.db.Keel(ctx)

	suppressed, err := suppressedKeys(ctx, db, req)
	if err != nil {
		return result, err
	}

	keep := make([]*insight.Insight, 0, len(req.Insights))
	for _, entity := range req.Insights {
		if _, isSuppressed := suppressed[entity.DedupeKey]; isSuppressed {
			result.Suppressed++
			continue
		}
		keep = append(keep, entity)
	}

	counts, err := applyReplacement(ctx, db, req, keep)
	if err != nil {
		return result, err
	}

	result.Superseded = counts.Superseded
	result.Resolved = counts.Resolved
	result.Created = counts.Created
	result.Detected = counts.Detected

	return result, nil
}

func suppressedKeys(
	ctx context.Context,
	db *keel.DB,
	req *insightports.ReplaceDetectorFindingsRequest,
) (map[string]struct{}, error) {
	in := insightdb.Insights
	keys, err := keel.Into[string](ctx, db, in.Pick(in.DedupeKey).Where(
		in.DetectorKey.Eq(req.DetectorKey),
		in.Status.Eq(insight.StatusDismissed),
		in.DismissedAt.Gte(req.SuppressedBefore),
	))
	if err != nil {
		return nil, fmt.Errorf("read suppressed findings: %w", err)
	}

	suppressed := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		suppressed[key] = struct{}{}
	}

	return suppressed, nil
}

func applyReplacement(
	ctx context.Context,
	db *keel.DB,
	req *insightports.ReplaceDetectorFindingsRequest,
	keep []*insight.Insight,
) (insightports.ReplaceDetectorFindingsResult, error) {
	var result insightports.ReplaceDetectorFindingsResult

	current := make([]string, 0, len(keep))
	for _, entity := range keep {
		current = append(current, entity.DedupeKey)
	}

	superseded, err := closeActive(ctx, db, closeParams{
		detectorKey: req.DetectorKey,
		status:      insight.StatusSuperseded,
		keys:        current,
		within:      true,
	})
	if err != nil {
		return result, err
	}
	result.Superseded = len(superseded)

	resolved, err := closeActive(ctx, db, closeParams{
		detectorKey: req.DetectorKey,
		status:      insight.StatusResolved,
		keys:        current,
		within:      false,
	})
	if err != nil {
		return result, err
	}
	result.Resolved = len(resolved)

	if len(keep) == 0 {
		return result, nil
	}

	if err = insightdb.Insights.CreateMany(ctx, db, keep); err != nil {
		return result, fmt.Errorf("insert findings: %w", err)
	}
	result.Created = len(keep)
	result.Detected = newlyDetected(keep, superseded)

	return result, nil
}

func newlyDetected(inserted []*insight.Insight, superseded []string) []*insight.Insight {
	seen := make(map[string]struct{}, len(superseded))
	for _, key := range superseded {
		seen[key] = struct{}{}
	}

	detected := make([]*insight.Insight, 0, len(inserted))
	for _, entity := range inserted {
		if _, known := seen[entity.DedupeKey]; !known {
			detected = append(detected, entity)
		}
	}

	return detected
}

type closeParams struct {
	detectorKey string
	status      insight.Status
	keys        []string
	within      bool
}

func closeActiveQuery(params closeParams) *keel.Query[*insight.Insight] {
	in := insightdb.Insights

	var keys keel.Pred
	switch {
	case len(params.keys) > 0 && params.within:
		keys = in.DedupeKey.In(params.keys...)
	case len(params.keys) > 0:
		keys = in.DedupeKey.NotIn(params.keys...)
	case params.within:
		return nil
	}

	return in.Pick(in.DedupeKey).Where(
		in.DetectorKey.Eq(params.detectorKey),
		in.Status.Eq(insight.StatusActive),
		keys,
	)
}

func closeActive(ctx context.Context, db *keel.DB, params closeParams) ([]string, error) {
	query := closeActiveQuery(params)
	if query == nil {
		return nil, nil
	}

	in := insightdb.Insights
	rows, err := query.UpdateReturning(ctx, db, in.Status.To(params.status), keepInsightVersion())
	if err != nil {
		return nil, fmt.Errorf("close active findings as %s: %w", params.status, err)
	}

	closed := make([]string, 0, len(rows))
	for _, row := range rows {
		closed = append(closed, row.DedupeKey)
	}

	return closed, nil
}
