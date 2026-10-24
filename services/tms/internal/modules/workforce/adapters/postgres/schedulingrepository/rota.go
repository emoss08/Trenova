package schedulingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerteam"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/jackc/pgx/v5"
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

// rotaAssignedDaysSQL reads the assignments table, which belongs to a module
// this one may not import, so the statement is written out with its tenant
// predicates. The roster is the same one rosterQuery builds: active workers,
// optionally of one fleet, optionally answering to the given managers. The day
// is the epoch truncated to its UTC midnight with integer arithmetic.
const rotaAssignedDaysSQL = `
SELECT asn.primary_worker_id AS worker_id,
	(asn.created_at / 86400) * 86400 AS day_start,
	COUNT(*)::int AS count
FROM assignments asn
JOIN workers w
	ON w.id = asn.primary_worker_id
	AND w.organization_id = asn.organization_id
	AND w.business_unit_id = asn.business_unit_id
WHERE asn.organization_id = $1
	AND asn.business_unit_id = $2
	AND asn.archived_at IS NULL
	AND (asn.created_at / 86400) * 86400 >= $3
	AND (asn.created_at / 86400) * 86400 < $4
	AND w.status = $5
	AND ($6 = '' OR w.fleet_code_id = $6)
	AND (
		cardinality($7::text[]) = 0
		OR w.manager_id = ANY($7::text[])
		OR w.fleet_code_id IN (
			SELECT fc.id
			FROM fleet_codes fc
			WHERE fc.organization_id = w.organization_id
				AND fc.business_unit_id = w.business_unit_id
				AND fc.manager_id = ANY($7::text[])
		)
	)
GROUP BY asn.primary_worker_id, (asn.created_at / 86400) * 86400`

func rosterFilter(req *workerports.RotaQuery) []keel.Pred {
	w := workerdb.Workers

	return []keel.Pred{
		w.Status.Eq(domaintypes.StatusActive),
		keel.When(req.FleetCodeID.IsNotNil(), w.FleetCodeID.Eq(req.FleetCodeID)),
		workerteam.Where(w.ID.Col, req.ManagerIDs),
	}
}

func rosterQuery(req *workerports.RotaQuery) *keel.Query[worker.Worker] {
	w := workerdb.Workers

	return w.Pick(w.ID).Where(rosterFilter(req)...)
}

func rotaWorkersQuery(req *workerports.RotaQuery) *keel.Query[worker.Worker] {
	w := workerdb.Workers
	fc := w.FleetCode()
	sa := workerdb.WorkerShiftAssignments

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
	req *workerports.RotaQuery,
) ([]workerports.RotaWorkerRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workerports.RotaWorkerRow, error) {
		rows, err := keel.Into[workerports.RotaWorkerRow](
			ctx,
			r.db.Keel(ctx),
			rotaWorkersQuery(req),
		)
		if err != nil {
			r.l.Error("failed to read rota workers", zap.Error(err))
			return nil, fmt.Errorf("read rota workers: %w", err)
		}

		return rows, nil
	})
}

// RotaTimeOffRanges is the approved time off overlapping the week. Ranges
// rather than days: the composer expands them.
func (r *repository) RotaTimeOffRanges(
	ctx context.Context,
	req *workerports.RotaQuery,
) ([]workerports.RotaRangeRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workerports.RotaRangeRow, error) {
		p := workerdb.WorkerPTOs
		rows, err := keel.Into[workerports.RotaRangeRow](ctx, r.db.Keel(ctx), p.
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
	})
}

// RotaLeaveRanges is the same over open leave cases. A case with no end date
// runs on, so it comes back with a zero end and the composer treats it as
// covering the rest of the week.
func (r *repository) RotaLeaveRanges(
	ctx context.Context,
	req *workerports.RotaQuery,
) ([]workerports.RotaRangeRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workerports.RotaRangeRow, error) {
		c := workerdb.WorkerLeaveCases
		rows, err := keel.Into[workerports.RotaRangeRow](ctx, r.db.Keel(ctx), c.
			Pick(c.WorkerID, c.StartsAt, c.EndsAt.Coalesce(int64(0))).
			Where(
				c.WorkerID.InQuery(rosterQuery(req)),
				c.Status.Eq(worker.LeaveCaseApproved),
				c.StartsAt.Lt(req.WeekEnd),
				keel.Or(c.EndsAt.IsNull(), c.EndsAt.Gte(req.WeekStart)),
			))
		if err != nil {
			r.l.Error("failed to read rota leave", zap.Error(err))
			return nil, fmt.Errorf("read rota leave: %w", err)
		}

		return rows, nil
	})
}

// RotaAssignedDays counts the work dispatch has already put on each day. The
// day is truncated with integer arithmetic on the epoch column.
func (r *repository) RotaAssignedDays(
	ctx context.Context,
	req *workerports.RotaQuery,
) ([]workerports.RotaDayRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workerports.RotaDayRow, error) {
		managers := make([]string, 0, len(req.ManagerIDs))
		for _, id := range req.ManagerIDs {
			managers = append(managers, id.String())
		}

		rows, err := keel.QueryAll(
			ctx,
			r.db.Keel(ctx),
			pgx.RowToStructByPos[workerports.RotaDayRow],
			rotaAssignedDaysSQL,
			req.TenantInfo.OrgID.String(),
			req.TenantInfo.BuID.String(),
			req.WeekStart,
			req.WeekEnd,
			string(domaintypes.StatusActive),
			req.FleetCodeID.String(),
			managers,
		)
		if err != nil {
			r.l.Error("failed to read rota assignments", zap.Error(err))
			return nil, fmt.Errorf("read rota assignments: %w", err)
		}

		return rows, nil
	})
}
