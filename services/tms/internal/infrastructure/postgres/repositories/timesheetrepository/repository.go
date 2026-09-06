package timesheetrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultEntryPageSize     = 500
	defaultTimesheetPageSize = 200
	defaultExportPageSize    = 100
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

func New(p Params) repositories.TimesheetRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.timesheet-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

func (r *repository) ListEntries(
	ctx context.Context,
	req *repositories.ListTimeClockEntriesRequest,
) ([]*worker.TimeClockEntry, error) {
	cols := buncolgen.TimeClockEntryColumns
	entities := make([]*worker.TimeClockEntry, 0, 16)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.TimeClockEntryScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if !req.TimesheetID.IsNil() {
				sq = sq.Where(cols.TimesheetID.Eq(), req.TimesheetID)
			}
			if req.From > 0 {
				sq = sq.Where(cols.ClockedInAt.Gte(), req.From)
			}
			if req.To > 0 {
				sq = sq.Where(cols.ClockedInAt.Lt(), req.To)
			}
			if req.OpenOnly {
				sq = sq.Where(cols.ClockedOutAt.IsNull())
			}
			return sq
		}).
		Order(cols.ClockedInAt.OrderAsc()).
		Limit(limitOr(req.Limit, defaultEntryPageSize)).
		Scan(ctx); err != nil {
		r.l.Error("failed to list time clock entries", zap.Error(err))
		return nil, fmt.Errorf("list time clock entries: %w", err)
	}

	return entities, nil
}

func (r *repository) GetEntryByID(
	ctx context.Context,
	req *repositories.GetTimeClockEntryByIDRequest,
) (*worker.TimeClockEntry, error) {
	entity := new(worker.TimeClockEntry)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TimeClockEntryScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.TimeClockEntryColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "TimeClockEntry")
	}

	return entity, nil
}

// GetOpenEntry is the punch a worker is currently on. A missing one is not an
// error — most of the time nobody is on the clock — so it comes back nil.
func (r *repository) GetOpenEntry(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.TimeClockEntry, error) {
	cols := buncolgen.TimeClockEntryColumns
	entity := new(worker.TimeClockEntry)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TimeClockEntryScopeTenant(sq, tenantInfo).
				Where(cols.WorkerID.Eq(), workerID).
				Where(cols.ClockedOutAt.IsNull())
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nobody on the clock is not an error
		}
		r.l.Error("failed to read open time clock entry", zap.Error(err))
		return nil, fmt.Errorf("read open time clock entry: %w", err)
	}

	return entity, nil
}

func (r *repository) CreateEntry(
	ctx context.Context,
	entity *worker.TimeClockEntry,
) (*worker.TimeClockEntry, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create time clock entry", zap.Error(err))
		return nil, fmt.Errorf("create time clock entry: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateEntry(
	ctx context.Context,
	entity *worker.TimeClockEntry,
) (*worker.TimeClockEntry, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.TimeClockEntryColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update time clock entry", zap.Error(err))
		return nil, fmt.Errorf("update time clock entry: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "TimeClockEntry", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) DeleteEntry(
	ctx context.Context,
	req *repositories.GetTimeClockEntryByIDRequest,
) error {
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*worker.TimeClockEntry)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.TimeClockEntryScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.TimeClockEntryColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete time clock entry", zap.Error(err))
		return fmt.Errorf("delete time clock entry: %w", err)
	}

	return dberror.CheckRowsAffected(results, "TimeClockEntry", req.ID.String())
}

func (r *repository) ListTimesheets(
	ctx context.Context,
	req *repositories.ListTimesheetsRequest,
) ([]*worker.Timesheet, error) {
	cols := buncolgen.TimesheetColumns
	entities := make([]*worker.Timesheet, 0, 16)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.TimesheetScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if len(req.Statuses) > 0 {
				sq = sq.Where(cols.Status.In(), bun.List(req.Statuses))
			}
			if req.PeriodStart > 0 {
				sq = sq.Where(cols.PeriodStart.Eq(), req.PeriodStart)
			}
			if req.From > 0 {
				sq = sq.Where(cols.PeriodStart.Gte(), req.From)
			}
			if req.To > 0 {
				sq = sq.Where(cols.PeriodStart.Lt(), req.To)
			}
			if !req.ExportID.IsNil() {
				sq = sq.Where(cols.PayrollExportID.Eq(), req.ExportID)
			}
			if req.UnexportedOnly {
				sq = sq.Where(cols.PayrollExportID.IsNull())
			}
			if len(req.ManagerIDs) > 0 {
				sq = sq.Where(
					"tsh.worker_id IN (SELECT id FROM workers"+
						" WHERE organization_id = tsh.organization_id"+
						" AND business_unit_id = tsh.business_unit_id"+
						" AND manager_id IN (?))",
					bun.List(req.ManagerIDs),
				)
			}
			return sq
		}).
		Order(cols.PeriodStart.OrderDesc()).
		Limit(limitOr(req.Limit, defaultTimesheetPageSize))

	if req.IncludeWorker {
		q = q.Relation(buncolgen.TimesheetRelations.Worker)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list timesheets", zap.Error(err))
		return nil, fmt.Errorf("list timesheets: %w", err)
	}

	return entities, nil
}

func (r *repository) GetTimesheetByID(
	ctx context.Context,
	req *repositories.GetTimesheetByIDRequest,
) (*worker.Timesheet, error) {
	entity := new(worker.Timesheet)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TimesheetScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.TimesheetColumns.ID.Eq(), req.ID)
		})
	if req.IncludeWorker {
		q = q.Relation(buncolgen.TimesheetRelations.Worker)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Timesheet")
	}

	if req.IncludeEntries {
		entries, err := r.ListEntries(ctx, &repositories.ListTimeClockEntriesRequest{
			TenantInfo:  req.TenantInfo,
			TimesheetID: entity.ID,
		})
		if err != nil {
			return nil, err
		}
		entity.Entries = entries
	}

	return entity, nil
}

// GetOrCreateTimesheet is the week a punch belongs to. Insert-on-conflict
// rather than read-then-write: two punches landing at once would otherwise both
// find no sheet and both try to make one.
func (r *repository) GetOrCreateTimesheet(
	ctx context.Context,
	entity *worker.Timesheet,
) (*worker.Timesheet, error) {
	db := r.db.DBForContext(ctx)

	if _, err := db.NewInsert().
		Model(entity).
		On("CONFLICT (organization_id, business_unit_id, worker_id, period_start) DO NOTHING").
		Exec(ctx); err != nil {
		r.l.Error("failed to open timesheet", zap.Error(err))
		return nil, fmt.Errorf("open timesheet: %w", err)
	}
	// Whichever side of the conflict this call was on, the row that exists is
	// the answer, so it is read back rather than assumed.
	existing := new(worker.Timesheet)
	if err := db.NewSelect().
		Model(existing).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TimesheetScopeTenant(sq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).
				Where(buncolgen.TimesheetColumns.WorkerID.Eq(), entity.WorkerID).
				Where(buncolgen.TimesheetColumns.PeriodStart.Eq(), entity.PeriodStart)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Timesheet")
	}

	return existing, nil
}

func (r *repository) UpdateTimesheet(
	ctx context.Context,
	entity *worker.Timesheet,
) (*worker.Timesheet, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.TimesheetColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update timesheet", zap.Error(err))
		return nil, fmt.Errorf("update timesheet: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "Timesheet", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

// AttachEntries rolls a week's loose punches onto the sheet they belong to. It
// takes only closed entries: a punch nobody has finished is not part of any
// week yet, and pulling it in would freeze a total that is still moving.
func (r *repository) AttachEntries(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timesheetID pulid.ID,
	workerID pulid.ID,
	from, to int64,
) (int, error) {
	cols := buncolgen.TimeClockEntryColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*worker.TimeClockEntry)(nil)).
		Set("timesheet_id = ?", timesheetID).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.TimeClockEntryScopeTenantUpdate(uq, tenantInfo).
				Where(cols.WorkerID.Eq(), workerID).
				Where(cols.ClockedInAt.Gte(), from).
				Where(cols.ClockedInAt.Lt(), to).
				Where(cols.ClockedOutAt.IsNotNull()).
				Where(cols.TimesheetID.IsNull())
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to attach time clock entries", zap.Error(err))
		return 0, fmt.Errorf("attach time clock entries: %w", err)
	}

	affected, err := results.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("attach time clock entries: %w", err)
	}

	return int(affected), nil
}

func (r *repository) ListExports(
	ctx context.Context,
	req *repositories.ListPayrollExportsRequest,
) ([]*worker.PayrollExport, error) {
	entities := make([]*worker.PayrollExport, 0, 8)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PayrollExportScopeTenant(sq, req.TenantInfo)
		}).
		Order(buncolgen.PayrollExportColumns.PeriodStart.OrderDesc()).
		Limit(limitOr(req.Limit, defaultExportPageSize)).
		Scan(ctx); err != nil {
		r.l.Error("failed to list payroll exports", zap.Error(err))
		return nil, fmt.Errorf("list payroll exports: %w", err)
	}

	return entities, nil
}

func (r *repository) GetExportByID(
	ctx context.Context,
	req *repositories.GetPayrollExportByIDRequest,
) (*worker.PayrollExport, error) {
	entity := new(worker.PayrollExport)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PayrollExportScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.PayrollExportColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "PayrollExport")
	}

	return entity, nil
}

func (r *repository) CreateExport(
	ctx context.Context,
	entity *worker.PayrollExport,
) (*worker.PayrollExport, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create payroll export", zap.Error(err))
		return nil, fmt.Errorf("create payroll export: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateExport(
	ctx context.Context,
	entity *worker.PayrollExport,
) (*worker.PayrollExport, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.PayrollExportColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update payroll export", zap.Error(err))
		return nil, fmt.Errorf("update payroll export: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "PayrollExport", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

// StampExport attaches a run to the sheets it carried and locks them in one
// statement. Locking them one at a time would leave a half-exported period if
// anything failed partway. The status guard is part of the statement rather
// than a prior read, so a sheet somebody reopened between the two is skipped
// instead of being locked out from under them.
func (r *repository) StampExport(
	ctx context.Context,
	req *repositories.StampExportRequest,
) (int, error) {
	cols := buncolgen.TimesheetColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*worker.Timesheet)(nil)).
		Set("payroll_export_id = ?", req.ExportID).
		Set("status = ?", req.Status).
		Set("version = version + 1").
		Set("updated_at = ?", bun.Safe(r.db.NowEpoch())).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.TimesheetScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.TimesheetIDs)).
				Where(cols.Status.Eq(), worker.TimesheetApproved).
				Where(cols.PayrollExportID.IsNull())
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to stamp payroll export", zap.Error(err))
		return 0, fmt.Errorf("stamp payroll export: %w", err)
	}

	affected, err := results.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("stamp payroll export: %w", err)
	}

	return int(affected), nil
}

// ApprovedLeaveRanges is the approved time off touching a window. Overlap
// rather than containment: a week off that starts on the Friday before still
// covers this week's Monday.
func (r *repository) ApprovedLeaveRanges(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	from, to int64,
) ([]repositories.RotaRangeRow, error) {
	cols := buncolgen.WorkerPTOColumns
	rows := make([]repositories.RotaRangeRow, 0, 4)

	if err := buncolgen.WorkerPTOScopeTenant(
		r.db.DBForContext(ctx).NewSelect().Model((*worker.WorkerPTO)(nil)),
		tenantInfo,
	).
		Where(cols.WorkerID.Eq(), workerID).
		Where(cols.Status.Eq(), worker.PTOStatusApproved).
		Where(cols.StartDate.Lt(), to).
		Where(cols.EndDate.Gte(), from).
		ColumnExpr(cols.WorkerID.As("worker_id")).
		ColumnExpr(cols.StartDate.As("starts_at")).
		ColumnExpr(cols.EndDate.As("ends_at")).
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to read approved leave", zap.Error(err))
		return nil, fmt.Errorf("read approved leave: %w", err)
	}

	return rows, nil
}

// ClearExport detaches every sheet from a voided run and puts them back to
// Approved, which is the state they were in when the run picked them up.
func (r *repository) ClearExport(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	exportID pulid.ID,
) (int, error) {
	cols := buncolgen.TimesheetColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*worker.Timesheet)(nil)).
		Set("payroll_export_id = NULL").
		Set("status = ?", worker.TimesheetApproved).
		Set("version = version + 1").
		Set("updated_at = ?", bun.Safe(r.db.NowEpoch())).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.TimesheetScopeTenantUpdate(uq, tenantInfo).
				Where(cols.PayrollExportID.Eq(), exportID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to clear payroll export", zap.Error(err))
		return 0, fmt.Errorf("clear payroll export: %w", err)
	}

	affected, err := results.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("clear payroll export: %w", err)
	}

	return int(affected), nil
}
