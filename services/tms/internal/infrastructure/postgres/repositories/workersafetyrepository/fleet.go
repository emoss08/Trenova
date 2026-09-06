package workersafetyrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// The fleet roll-up is grouped SQL rather than a walk of the roster. Every
// number on the page is an aggregate, and pulling a few thousand drivers into
// memory to count them would be the wrong shape at any fleet size.

// outOfServiceCount counts the orders an event carries. The flag and the
// inspection result both record one, and an inspection that put a driver out
// of service sets both, so the OR is what stops it being counted twice.
const outOfServiceCount = "COUNT(*) FILTER (WHERE wsev.out_of_service" +
	" OR wsev.inspection_result = 'OutOfService') AS out_of_service"

// rosterScope is the roster half of every fleet query: active workers, their
// cached safety standing, and the terminal they belong to. The cache is what
// the roster itself shows, so the fleet view and the list behind it cannot
// disagree.
func (r *repository) rosterScope(
	ctx context.Context,
	req *repositories.FleetSafetyRequest,
) *bun.SelectQuery {
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.Worker)(nil)).
		Join("JOIN worker_profiles AS wrkp").
		JoinOn("wrkp.worker_id = wrk.id").
		JoinOn("wrkp.organization_id = wrk.organization_id").
		JoinOn("wrkp.business_unit_id = wrk.business_unit_id").
		Where("wrk.organization_id = ?", req.TenantInfo.OrgID).
		Where("wrk.business_unit_id = ?", req.TenantInfo.BuID).
		Where("wrk.status = ?", "Active")
	if !req.FleetCodeID.IsNil() {
		q = q.Where("wrk.fleet_code_id = ?", req.FleetCodeID)
	}
	return q
}

// eventScope is the event half: events inside the window, for active workers
// in the requested terminal.
func (r *repository) eventScope(
	ctx context.Context,
	req *repositories.FleetSafetyRequest,
	since int64,
) *bun.SelectQuery {
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerSafetyEvent)(nil)).
		Join("JOIN workers AS wrkr").
		JoinOn("wrkr.id = wsev.worker_id").
		JoinOn("wrkr.organization_id = wsev.organization_id").
		JoinOn("wrkr.business_unit_id = wsev.business_unit_id").
		Where("wsev.organization_id = ?", req.TenantInfo.OrgID).
		Where("wsev.business_unit_id = ?", req.TenantInfo.BuID).
		Where("wsev.occurred_at >= ?", since)
	if !req.FleetCodeID.IsNil() {
		q = q.Where("wrkr.fleet_code_id = ?", req.FleetCodeID)
	}
	return q
}

func (r *repository) FleetRatings(
	ctx context.Context,
	req *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyRatingRow, error) {
	rows := make([]repositories.FleetSafetyRatingRow, 0, 4)
	if err := r.rosterScope(ctx, req).
		ColumnExpr("wrkp.safety_rating AS rating").
		ColumnExpr("COUNT(*) AS workers").
		ColumnExpr("COALESCE(SUM(wrkp.safety_score), 0) AS total_score").
		GroupExpr("wrkp.safety_rating").
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to count fleet safety ratings", zap.Error(err))
		return nil, fmt.Errorf("count fleet safety ratings: %w", err)
	}
	return rows, nil
}

func (r *repository) FleetTerminals(
	ctx context.Context,
	req *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyTerminalRow, error) {
	rows := make([]repositories.FleetSafetyTerminalRow, 0, 8)
	if err := r.rosterScope(ctx, req).
		Join("LEFT JOIN fleet_codes AS fc").
		JoinOn("fc.id = wrk.fleet_code_id").
		JoinOn("fc.organization_id = wrk.organization_id").
		JoinOn("fc.business_unit_id = wrk.business_unit_id").
		ColumnExpr("COALESCE(wrk.fleet_code_id, '') AS fleet_code_id").
		ColumnExpr("COALESCE(fc.code, '') AS fleet_code_code").
		ColumnExpr("COALESCE(fc.description, '') AS fleet_code_description").
		ColumnExpr("COALESCE(fc.color, '') AS fleet_code_color").
		ColumnExpr("COUNT(*) AS workers").
		ColumnExpr("COUNT(*) FILTER (WHERE wrkp.safety_rating = 'AtRisk') AS at_risk").
		ColumnExpr("COUNT(*) FILTER (WHERE wrkp.safety_rating = 'Watch') AS watch").
		ColumnExpr("COALESCE(SUM(wrkp.safety_score), 0) AS total_score").
		GroupExpr("wrk.fleet_code_id, fc.code, fc.description, fc.color").
		OrderExpr("at_risk DESC, workers DESC").
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to group fleet safety by terminal", zap.Error(err))
		return nil, fmt.Errorf("group fleet safety by terminal: %w", err)
	}
	return rows, nil
}

func (r *repository) FleetKinds(
	ctx context.Context,
	req *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyKindRow, error) {
	since := timeutils.AddMonthsUTC(req.Now, -req.WindowMonths)
	rows := make([]repositories.FleetSafetyKindRow, 0, 5)
	if err := r.eventScope(ctx, req, since).
		ColumnExpr("wsev.kind AS kind").
		ColumnExpr("COUNT(*) AS events").
		ColumnExpr("COALESCE(SUM(wsev.points), 0) AS points").
		ColumnExpr("COUNT(*) FILTER (WHERE wsev.preventable) AS preventable").
		ColumnExpr(outOfServiceCount).
		ColumnExpr("COUNT(*) FILTER (WHERE wsev.status <> 'Closed') AS open_events").
		GroupExpr("wsev.kind").
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to group fleet safety by kind", zap.Error(err))
		return nil, fmt.Errorf("group fleet safety by kind: %w", err)
	}
	return rows, nil
}

// csaBucketExpr sorts an event into the FMCSA's recency buckets: 0 inside six
// months, 1 inside a year, 2 out to the two-year look-back. The weights
// themselves stay in the domain, where they can be tested without a database.
func csaBucketExpr(now int64) string {
	return fmt.Sprintf(
		"CASE WHEN wsev.occurred_at >= %d THEN 0 WHEN wsev.occurred_at >= %d THEN 1 ELSE 2 END",
		timeutils.AddMonthsUTC(now, -6),
		timeutils.AddMonthsUTC(now, -12),
	)
}

func (r *repository) FleetBasics(
	ctx context.Context,
	req *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyBasicRow, error) {
	since := timeutils.AddMonthsUTC(req.Now, -worker.SafetyPointsRetentionMonths)
	rows := make([]repositories.FleetSafetyBasicRow, 0, 16)
	bucket := csaBucketExpr(req.Now)

	if err := r.eventScope(ctx, req, since).
		Join("JOIN worker_safety_violations AS wsvi").
		JoinOn("wsvi.safety_event_id = wsev.id").
		JoinOn("wsvi.organization_id = wsev.organization_id").
		JoinOn("wsvi.business_unit_id = wsev.business_unit_id").
		ColumnExpr("wsvi.basic AS basic").
		ColumnExpr(bucket+" AS bucket").
		ColumnExpr("COUNT(*) AS violations").
		ColumnExpr("COALESCE(SUM(wsvi.severity_weight), 0) AS severity_sum").
		ColumnExpr("COUNT(*) FILTER (WHERE wsvi.out_of_service) AS out_of_service").
		GroupExpr("wsvi.basic, "+bucket).
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to group fleet safety by BASIC", zap.Error(err))
		return nil, fmt.Errorf("group fleet safety by BASIC: %w", err)
	}
	return rows, nil
}

// FleetEventBasics is the fallback for events nobody keyed violations into.
// Without it a carrier who records events but not violation codes would read a
// blank scorecard, which is worse than an approximate one.
func (r *repository) FleetEventBasics(
	ctx context.Context,
	req *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyEventBasicRow, error) {
	since := timeutils.AddMonthsUTC(req.Now, -worker.SafetyPointsRetentionMonths)
	rows := make([]repositories.FleetSafetyEventBasicRow, 0, 16)
	bucket := csaBucketExpr(req.Now)

	if err := r.eventScope(ctx, req, since).
		Where(
			"NOT EXISTS (SELECT 1 FROM worker_safety_violations wsvi"+
				" WHERE wsvi.safety_event_id = wsev.id"+
				" AND wsvi.organization_id = wsev.organization_id"+
				" AND wsvi.business_unit_id = wsev.business_unit_id)",
		).
		ColumnExpr("wsev.kind AS kind").
		ColumnExpr("COALESCE(wsev.inspection_result, '') AS inspection_result").
		ColumnExpr(bucket+" AS bucket").
		ColumnExpr("COUNT(*) AS events").
		ColumnExpr("COALESCE(SUM(wsev.points), 0) AS points").
		ColumnExpr(outOfServiceCount).
		GroupExpr("wsev.kind, wsev.inspection_result, "+bucket).
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to group fleet safety events by BASIC", zap.Error(err))
		return nil, fmt.Errorf("group fleet safety events by BASIC: %w", err)
	}
	return rows, nil
}

func (r *repository) FleetTrend(
	ctx context.Context,
	req *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyTrendRow, error) {
	since := timeutils.AddMonthsUTC(req.Now, -req.WindowMonths)
	rows := make([]repositories.FleetSafetyTrendRow, 0, 24)

	// Bucketing by UTC month in SQL keeps the trend a single scan. Months are
	// what a safety meeting talks in, and the day boundary an org's timezone
	// would move is not worth a second query. The dialect spells the truncation
	// so the query is not Postgres-only.
	period := dbdialect.MonthStartEpochFromBun(r.db.DBForContext(ctx), "wsev.occurred_at")

	if err := r.eventScope(ctx, req, since).
		ColumnExpr(period+" AS period_start").
		ColumnExpr("COUNT(*) AS events").
		ColumnExpr("COUNT(*) FILTER (WHERE wsev.kind = 'Accident') AS accidents").
		ColumnExpr(
			"COUNT(*) FILTER (WHERE wsev.kind = 'Accident' AND wsev.preventable) AS preventable",
		).
		ColumnExpr("COUNT(*) FILTER (WHERE wsev.kind = 'Citation') AS citations").
		ColumnExpr("COUNT(*) FILTER (WHERE wsev.kind = 'Inspection') AS inspections").
		ColumnExpr(outOfServiceCount).
		ColumnExpr("COALESCE(SUM(wsev.points), 0) AS points").
		GroupExpr(period).
		OrderExpr("period_start").
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to build fleet safety trend", zap.Error(err))
		return nil, fmt.Errorf("build fleet safety trend: %w", err)
	}
	return rows, nil
}

func (r *repository) FleetRanking(
	ctx context.Context,
	req *repositories.FleetSafetyRequest,
) ([]repositories.FleetSafetyRankRow, error) {
	since := timeutils.AddMonthsUTC(req.Now, -req.WindowMonths)
	rows := make([]repositories.FleetSafetyRankRow, 0, 32)

	// Active points are not cached on the profile — they roll off on a date
	// rather than on a write — so they are summed here, once, over the same
	// scan that produces the ranking.
	if err := r.rosterScope(ctx, req).
		Join("LEFT JOIN fleet_codes AS fc").
		JoinOn("fc.id = wrk.fleet_code_id").
		JoinOn("fc.organization_id = wrk.organization_id").
		JoinOn("fc.business_unit_id = wrk.business_unit_id").
		Join("LEFT JOIN worker_safety_events AS wsev").
		JoinOn("wsev.worker_id = wrk.id").
		JoinOn("wsev.organization_id = wrk.organization_id").
		JoinOn("wsev.business_unit_id = wrk.business_unit_id").
		ColumnExpr("wrk.id AS worker_id").
		ColumnExpr("wrk.first_name AS first_name").
		ColumnExpr("wrk.last_name AS last_name").
		ColumnExpr("COALESCE(wrk.fleet_code_id, '') AS fleet_code_id").
		ColumnExpr("COALESCE(fc.code, '') AS fleet_code_code").
		ColumnExpr("COALESCE(fc.color, '') AS fleet_code_color").
		ColumnExpr("wrkp.safety_rating AS rating").
		ColumnExpr("wrkp.safety_score AS score").
		ColumnExpr(
			"COALESCE(SUM(wsev.points) FILTER (WHERE wsev.points > 0"+
				" AND (wsev.points_expire_at IS NULL OR wsev.points_expire_at > ?)), 0) AS active_points",
			req.Now,
		).
		ColumnExpr("COUNT(wsev.id) FILTER (WHERE wsev.occurred_at >= ?) AS events", since).
		ColumnExpr(
			"MAX(wsev.occurred_at) FILTER (WHERE wsev.kind IN ('Accident', 'Incident', 'Citation')) AS last_event_at",
		).
		GroupExpr(
			"wrk.id, wrk.first_name, wrk.last_name, wrk.fleet_code_id,"+
				" fc.code, fc.color, wrkp.safety_rating, wrkp.safety_score",
		).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			if req.RankBest {
				return sq.OrderExpr("score DESC, active_points ASC, last_name ASC")
			}
			return sq.OrderExpr("score ASC, active_points DESC, last_name ASC")
		}).
		Limit(req.RankLimit).
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to rank fleet safety", zap.Error(err))
		return nil, fmt.Errorf("rank fleet safety: %w", err)
	}
	return rows, nil
}
