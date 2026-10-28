package workersafetyrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety/workersafetydb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workersafetyports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// The fleet roll-up is grouped SQL rather than a walk of the roster. Every
// number on the page is an aggregate, and pulling a few thousand drivers into
// memory to count them would be the wrong shape at any fleet size.

// outOfServiceCount counts the orders an event carries. The flag and the
// inspection result both record one, and an inspection that put a driver out
// of service sets both, so the OR is what stops it being counted twice.
func outOfServiceCount() keel.Agg {
	e := workersafetydb.WorkerSafetyEvents

	return keel.Count().Filter(keel.Or(
		e.OutOfService.IsTrue(),
		e.InspectionResult.Eq(workersafety.InspectionResultOutOfService),
	))
}

func sumOrZero(e keel.Expr) keel.Int64Expr {
	return keel.Int64Of(e).Coalesce(0)
}

// rosterScope is the roster half of every fleet query: active workers, their
// cached safety standing, and the terminal they belong to. The cache is what
// the roster itself shows, so the fleet view and the list behind it cannot
// disagree.
func rosterScope(req *workersafetyports.FleetSafetyRequest) *keel.Query[*worker.Worker] {
	w := workerdb.Workers
	wp := workerdb.WorkerProfiles

	return w.Join(wp, wp.WorkerID.EqCol(w.ID)).Where(
		w.Status.Eq(domaintypes.StatusActive),
		keel.When(req.FleetCodeID.IsNotNil(), w.FleetCodeID.Eq(req.FleetCodeID)),
	)
}

// eventScope is the event half: events inside the window, for active workers
// in the requested terminal.
func eventScope(
	req *workersafetyports.FleetSafetyRequest,
	since int64,
) *keel.Query[*workersafety.WorkerSafetyEvent] {
	e := workersafetydb.WorkerSafetyEvents
	w := workerdb.Workers

	return e.Join(w, w.ID.EqCol(e.WorkerID)).Where(
		e.OccurredAt.Gte(since),
		keel.When(req.FleetCodeID.IsNotNil(), w.FleetCodeID.Eq(req.FleetCodeID)),
	)
}

func fleetRatingsQuery(req *workersafetyports.FleetSafetyRequest) *keel.Query[*worker.Worker] {
	wp := workerdb.WorkerProfiles

	return rosterScope(req).
		Group(wp.SafetyRating).
		Pick(wp.SafetyRating, keel.Count(), sumOrZero(wp.SafetyScore.Sum()))
}

func (r *repository) FleetRatings(
	ctx context.Context,
	req *workersafetyports.FleetSafetyRequest,
) ([]workersafetyports.FleetSafetyRatingRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workersafetyports.FleetSafetyRatingRow, error) {
			rows, err := keel.Into[workersafetyports.FleetSafetyRatingRow](
				ctx,
				r.db.Keel(ctx),
				fleetRatingsQuery(req),
			)
			if err != nil {
				r.l.Error("failed to count fleet safety ratings", zap.Error(err))
				return nil, fmt.Errorf("count fleet safety ratings: %w", err)
			}

			return rows, nil
		},
	)
}

func fleetTerminalsQuery(req *workersafetyports.FleetSafetyRequest) *keel.Query[*worker.Worker] {
	w := workerdb.Workers
	wp := workerdb.WorkerProfiles
	fc := w.FleetCode()
	atRisk := keel.Count().Filter(wp.SafetyRating.Eq(worker.SafetyRatingAtRisk))

	return rosterScope(req).
		Group(w.FleetCodeID, fc.Code, fc.Description, fc.Color).
		Pick(
			w.FleetCodeID.Coalesce(""),
			fc.Code.Coalesce(""),
			fc.Description.Coalesce(""),
			fc.Color.Coalesce(""),
			keel.Count(),
			atRisk,
			keel.Count().Filter(wp.SafetyRating.Eq(worker.SafetyRatingWatch)),
			sumOrZero(wp.SafetyScore.Sum()),
		).
		Order(atRisk.Desc(), keel.Count().Desc())
}

func (r *repository) FleetTerminals(
	ctx context.Context,
	req *workersafetyports.FleetSafetyRequest,
) ([]workersafetyports.FleetSafetyTerminalRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workersafetyports.FleetSafetyTerminalRow, error) {
			rows, err := keel.Into[workersafetyports.FleetSafetyTerminalRow](
				ctx,
				r.db.Keel(ctx),
				fleetTerminalsQuery(req),
			)
			if err != nil {
				r.l.Error("failed to group fleet safety by terminal", zap.Error(err))
				return nil, fmt.Errorf("group fleet safety by terminal: %w", err)
			}

			return rows, nil
		},
	)
}

func (r *repository) FleetKinds(
	ctx context.Context,
	req *workersafetyports.FleetSafetyRequest,
) ([]workersafetyports.FleetSafetyKindRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workersafetyports.FleetSafetyKindRow, error) {
			e := workersafetydb.WorkerSafetyEvents
			since := timeutils.AddMonthsUTC(req.Now, -req.WindowMonths)
			rows, err := keel.Into[workersafetyports.FleetSafetyKindRow](ctx, r.db.Keel(ctx),
				eventScope(req, since).
					Group(e.Kind).
					Pick(
						e.Kind,
						keel.Count(),
						sumOrZero(e.Points.Sum()),
						keel.Count().Filter(e.Preventable.IsTrue()),
						outOfServiceCount(),
						keel.Count().Filter(e.Status.Ne(workersafety.SafetyEventStatusClosed)),
					))
			if err != nil {
				r.l.Error("failed to group fleet safety by kind", zap.Error(err))
				return nil, fmt.Errorf("group fleet safety by kind: %w", err)
			}

			return rows, nil
		},
	)
}

// csaBucketExpr sorts an event into the FMCSA's recency buckets: 0 inside six
// months, 1 inside a year, 2 out to the two-year look-back. The weights
// themselves stay in the domain, where they can be tested without a database.
func csaBucketExpr(now int64) keel.Int64Expr {
	e := workersafetydb.WorkerSafetyEvents

	return keel.CaseInt64().
		When(e.OccurredAt.Gte(timeutils.AddMonthsUTC(now, -6)), keel.Int64(0)).
		When(e.OccurredAt.Gte(timeutils.AddMonthsUTC(now, -12)), keel.Int64(1)).
		Else(keel.Int64(2))
}

func fleetBasicsQuery(
	req *workersafetyports.FleetSafetyRequest,
) *keel.Query[*workersafety.WorkerSafetyEvent] {
	e := workersafetydb.WorkerSafetyEvents
	v := workersafetydb.WorkerSafetyViolations
	since := timeutils.AddMonthsUTC(req.Now, -workersafety.SafetyPointsRetentionMonths)
	bucket := csaBucketExpr(req.Now)

	return eventScope(req, since).
		Join(v, v.SafetyEventID.EqCol(e.ID)).
		Group(v.Basic, bucket).
		Pick(
			v.Basic,
			bucket,
			keel.Count(),
			sumOrZero(v.SeverityWeight.Sum()),
			keel.Count().Filter(v.OutOfService.IsTrue()),
		)
}

func (r *repository) FleetBasics(
	ctx context.Context,
	req *workersafetyports.FleetSafetyRequest,
) ([]workersafetyports.FleetSafetyBasicRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workersafetyports.FleetSafetyBasicRow, error) {
			rows, err := keel.Into[workersafetyports.FleetSafetyBasicRow](
				ctx,
				r.db.Keel(ctx),
				fleetBasicsQuery(req),
			)
			if err != nil {
				r.l.Error("failed to group fleet safety by BASIC", zap.Error(err))
				return nil, fmt.Errorf("group fleet safety by BASIC: %w", err)
			}

			return rows, nil
		},
	)
}

// FleetEventBasics is the fallback for events nobody keyed violations into.
// Without it a carrier who records events but not violation codes would read a
// blank scorecard, which is worse than an approximate one.
func (r *repository) FleetEventBasics(
	ctx context.Context,
	req *workersafetyports.FleetSafetyRequest,
) ([]workersafetyports.FleetSafetyEventBasicRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workersafetyports.FleetSafetyEventBasicRow, error) {
			e := workersafetydb.WorkerSafetyEvents
			v := workersafetydb.WorkerSafetyViolations
			since := timeutils.AddMonthsUTC(req.Now, -workersafety.SafetyPointsRetentionMonths)
			bucket := csaBucketExpr(req.Now)

			rows, err := keel.Into[workersafetyports.FleetSafetyEventBasicRow](ctx, r.db.Keel(ctx),
				eventScope(req, since).
					Where(keel.NotExists(v.Where(v.SafetyEventID.EqCol(e.ID)))).
					Group(e.Kind, e.InspectionResult, bucket).
					Pick(
						e.Kind,
						keel.CastText(e.InspectionResult).Coalesce(""),
						bucket,
						keel.Count(),
						sumOrZero(e.Points.Sum()),
						outOfServiceCount(),
					))
			if err != nil {
				r.l.Error("failed to group fleet safety events by BASIC", zap.Error(err))
				return nil, fmt.Errorf("group fleet safety events by BASIC: %w", err)
			}

			return rows, nil
		},
	)
}

func fleetTrendQuery(
	req *workersafetyports.FleetSafetyRequest,
) *keel.Query[*workersafety.WorkerSafetyEvent] {
	e := workersafetydb.WorkerSafetyEvents
	since := timeutils.AddMonthsUTC(req.Now, -req.WindowMonths)
	period := keel.Epoch(
		keel.DateTrunc("month", keel.ToTimestamp(e.OccurredAt).AtTimeZone(keel.Text("UTC"))),
	)
	accident := e.Kind.Eq(workersafety.SafetyEventAccident)

	return eventScope(req, since).
		Group(period).
		Pick(
			period,
			keel.Count(),
			keel.Count().Filter(accident),
			keel.Count().Filter(keel.And(accident, e.Preventable.IsTrue())),
			keel.Count().Filter(e.Kind.Eq(workersafety.SafetyEventCitation)),
			keel.Count().Filter(e.Kind.Eq(workersafety.SafetyEventInspection)),
			outOfServiceCount(),
			sumOrZero(e.Points.Sum()),
		).
		Order(keel.Asc(period))
}

func (r *repository) FleetTrend(
	ctx context.Context,
	req *workersafetyports.FleetSafetyRequest,
) ([]workersafetyports.FleetSafetyTrendRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workersafetyports.FleetSafetyTrendRow, error) {
			// Bucketing by UTC month in SQL keeps the trend a single scan. Months are
			// what a safety meeting talks in, and the day boundary an org's timezone
			// would move is not worth a second query.
			rows, err := keel.Into[workersafetyports.FleetSafetyTrendRow](
				ctx,
				r.db.Keel(ctx),
				fleetTrendQuery(req),
			)
			if err != nil {
				r.l.Error("failed to build fleet safety trend", zap.Error(err))
				return nil, fmt.Errorf("build fleet safety trend: %w", err)
			}

			return rows, nil
		},
	)
}

func fleetRankingQuery(req *workersafetyports.FleetSafetyRequest) *keel.Query[*worker.Worker] {
	w := workerdb.Workers
	wp := workerdb.WorkerProfiles
	e := workersafetydb.WorkerSafetyEvents
	fc := w.FleetCode()
	since := timeutils.AddMonthsUTC(req.Now, -req.WindowMonths)

	activePoints := sumOrZero(e.Points.Sum().Filter(keel.And(
		e.Points.Gt(0),
		keel.Or(e.PointsExpireAt.IsNull(), e.PointsExpireAt.Gt(req.Now)),
	)))

	order := []keel.Order{wp.SafetyScore.Asc(), keel.Desc(activePoints), w.LastName.Asc()}
	if req.RankBest {
		order = []keel.Order{wp.SafetyScore.Desc(), keel.Asc(activePoints), w.LastName.Asc()}
	}

	q := rosterScope(req).
		LeftJoin(e, e.WorkerID.EqCol(w.ID)).
		Group(
			w.ID,
			w.FirstName,
			w.LastName,
			w.FleetCodeID,
			fc.Code,
			fc.Color,
			wp.SafetyRating,
			wp.SafetyScore,
		).
		Pick(
			w.ID,
			w.FirstName,
			w.LastName,
			w.FleetCodeID.Coalesce(""),
			fc.Code.Coalesce(""),
			fc.Color.Coalesce(""),
			wp.SafetyRating,
			wp.SafetyScore,
			activePoints,
			keel.CountOf(e.ID).Filter(e.OccurredAt.Gte(since)),
			e.OccurredAt.Max().Filter(e.Kind.In(
				workersafety.SafetyEventAccident,
				workersafety.SafetyEventIncident,
				workersafety.SafetyEventCitation,
			)),
		).
		Order(order...)
	if req.RankLimit > 0 {
		q = q.Limit(req.RankLimit)
	}

	return q
}

func (r *repository) FleetRanking(
	ctx context.Context,
	req *workersafetyports.FleetSafetyRequest,
) ([]workersafetyports.FleetSafetyRankRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workersafetyports.FleetSafetyRankRow, error) {
			// Active points are not cached on the profile — they roll off on a date
			// rather than on a write — so they are summed here, once, over the same
			// scan that produces the ranking.
			rows, err := keel.Into[workersafetyports.FleetSafetyRankRow](
				ctx,
				r.db.Keel(ctx),
				fleetRankingQuery(req),
			)
			if err != nil {
				r.l.Error("failed to rank fleet safety", zap.Error(err))
				return nil, fmt.Errorf("rank fleet safety: %w", err)
			}

			return rows, nil
		},
	)
}
