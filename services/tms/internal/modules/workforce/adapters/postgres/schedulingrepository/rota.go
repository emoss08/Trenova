package schedulingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerteam"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave/workerleavedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule/workerscheduledb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerscheduleports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const defaultRotaPageSize = 300

// The rota is four grouped queries over a week rather than a walk of the
// roster: a fifty-driver week is four round trips, not two hundred.
//
// Time off and leave come back as the ranges they are stored as and are
// expanded into days by the composer. Expanding them in SQL would need
// generate_series, and a week is seven days — the expansion is not the
// expensive part.

func rosterFilter(req *workerscheduleports.RotaQuery) []keel.Pred {
	w := workerdb.Workers

	return []keel.Pred{
		w.Status.Eq(domaintypes.StatusActive),
		keel.When(req.FleetCodeID.IsNotNil(), w.FleetCodeID.Eq(req.FleetCodeID)),
		workerteam.Where(w.ID.Col, req.ManagerIDs),
	}
}

func rosterQuery(req *workerscheduleports.RotaQuery) *keel.Query[*worker.Worker] {
	w := workerdb.Workers

	return w.Pick(w.ID).Where(rosterFilter(req)...)
}

func rotaWorkersQuery(req *workerscheduleports.RotaQuery) *keel.Query[*worker.Worker] {
	w := workerdb.Workers
	fc := w.FleetCode()
	sa := workerscheduledb.WorkerShiftAssignments

	return w.LeftJoin(sa,
		sa.WorkerID.EqCol(w.ID),
		sa.EffectiveFrom.Lte(req.WeekEnd),
		keel.Or(sa.EffectiveTo.IsNull(), sa.EffectiveTo.Gte(req.WeekStart)),
	).
		Where(rosterFilter(req)...).
		Pick(
			w.ID,
			w.FirstName,
			w.LastName,
			fc.Code.Coalesce(""),
			fc.Color.Coalesce(""),
			sa.ShiftTemplateID.Coalesce(""),
			sa.CycleOffsetWeeks.Coalesce(int16(0)),
		).
		Order(w.LastName.Asc(), w.FirstName.Asc()).
		Limit(limitOr(req.Limit, defaultRotaPageSize))
}

// RotaWorkers is the roster line: who is on the board, and which pattern they
// are on for the week. The assignment in force is joined rather than listed
// separately so the week comes back in one row per worker.
func (r *repository) RotaWorkers(
	ctx context.Context,
	req *workerscheduleports.RotaQuery,
) ([]workerscheduleports.RotaWorkerRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workerscheduleports.RotaWorkerRow, error) {
			rows, err := keel.Into[workerscheduleports.RotaWorkerRow](
				ctx,
				r.db.Keel(ctx),
				rotaWorkersQuery(req),
			)
			if err != nil {
				r.l.Error("failed to read rota workers", zap.Error(err))
				return nil, fmt.Errorf("read rota workers: %w", err)
			}

			return rows, nil
		},
	)
}

// RotaTimeOffRanges is the approved time off overlapping the week. Ranges
// rather than days: the composer expands them.
func (r *repository) RotaTimeOffRanges(
	ctx context.Context,
	req *workerscheduleports.RotaQuery,
) ([]workerscheduleports.RotaRangeRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workerscheduleports.RotaRangeRow, error) {
			p := workerdb.WorkerPTOs
			rows, err := keel.Into[workerscheduleports.RotaRangeRow](ctx, r.db.Keel(ctx), p.
				Pick(p.WorkerID, p.StartDate, p.EndDate).
				Where(
					p.WorkerID.InQuery(rosterQuery(req)),
					p.Status.Eq(worker.PTOStatusApproved),
					p.StartDate.Lt(req.WeekEnd),
					p.EndDate.Gte(req.WeekStart),
				))
			if err != nil {
				r.l.Error("failed to read rota time off", zap.Error(err))
				return nil, fmt.Errorf("read rota time off: %w", err)
			}

			return rows, nil
		},
	)
}

// RotaLeaveRanges is the same over open leave cases. A case with no end date
// runs on, so it comes back with a zero end and the composer treats it as
// covering the rest of the week.
func (r *repository) RotaLeaveRanges(
	ctx context.Context,
	req *workerscheduleports.RotaQuery,
) ([]workerscheduleports.RotaRangeRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workerscheduleports.RotaRangeRow, error) {
			c := workerleavedb.WorkerLeaveCases
			rows, err := keel.Into[workerscheduleports.RotaRangeRow](ctx, r.db.Keel(ctx), c.
				Pick(c.WorkerID, c.StartsAt, c.EndsAt.Coalesce(int64(0))).
				Where(
					c.WorkerID.InQuery(rosterQuery(req)),
					c.Status.Eq(workerleave.LeaveCaseApproved),
					c.StartsAt.Lt(req.WeekEnd),
					keel.Or(c.EndsAt.IsNull(), c.EndsAt.Gte(req.WeekStart)),
				))
			if err != nil {
				r.l.Error("failed to read rota leave", zap.Error(err))
				return nil, fmt.Errorf("read rota leave: %w", err)
			}

			return rows, nil
		},
	)
}

func (r *repository) assignedDaysQuery(req *workerscheduleports.RotaQuery) *keel.Query[*keel.Row] {
	a := r.assignments
	secondsPerDay := keel.ConstInt64(timeutils.SecondsPerDay)
	day := a.CreatedAt.Div(secondsPerDay).Mul(secondsPerDay)

	return keel.Handle[*keel.Row]{Ref: a.WorkerID.Ref}.
		Pick(a.WorkerID, day, keel.Count()).
		Where(
			a.ArchivedAt.IsNull(),
			a.CreatedAt.Gte(firstDayStartFrom(req.WeekStart)),
			a.CreatedAt.Lt(firstDayStartFrom(req.WeekEnd)),
			a.WorkerID.InQuery(rosterQuery(req)),
		).
		Group(a.WorkerID, day)
}

func firstDayStartFrom(at int64) int64 {
	start := timeutils.DayStartUTC(at)
	if start < at {
		start += timeutils.SecondsPerDay
	}

	return start
}

// RotaAssignedDays counts the work dispatch has already put on each day. The
// day is truncated with integer arithmetic on the epoch column.
func (r *repository) RotaAssignedDays(
	ctx context.Context,
	req *workerscheduleports.RotaQuery,
) ([]workerscheduleports.RotaDayRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workerscheduleports.RotaDayRow, error) {
			rows, err := keel.Into[workerscheduleports.RotaDayRow](
				ctx,
				r.db.Keel(ctx),
				r.assignedDaysQuery(req),
			)
			if err != nil {
				r.l.Error("failed to read rota assignments", zap.Error(err))
				return nil, fmt.Errorf("read rota assignments: %w", err)
			}

			return rows, nil
		},
	)
}
