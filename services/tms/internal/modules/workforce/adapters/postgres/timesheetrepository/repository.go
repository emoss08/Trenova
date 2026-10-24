package timesheetrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerteam"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultEntryPageSize     = 500
	defaultTimesheetPageSize = 200
	defaultExportPageSize    = 100
)

var entryErrors = dberror.Mapping{
	Entity: "TimeClockEntry",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_time_clock_entries_open",
			Field:   "workerId",
			Message: "Worker is already on the clock",
		},
	},
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_time_clock_entries_worker",
			Field:   "workerId",
			Message: "Worker does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_time_clock_entries_timesheet",
			Field:   "timesheetId",
			Message: "Timesheet does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_time_clock_entries_pay_code",
			Field:   "payCodeId",
			Message: "Pay code does not exist or belongs to a different organization",
		},
	},
}

var timesheetErrors = dberror.Mapping{
	Entity: "Timesheet",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_timesheets_worker_period",
			Field:   "periodStart",
			Message: "Worker already has a timesheet for this period",
		},
	},
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_timesheets_worker",
			Field:   "workerId",
			Message: "Worker does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_timesheets_payroll_export",
			Field:   "payrollExportId",
			Message: "Payroll export does not exist or belongs to a different organization",
		},
	},
}

var exportErrors = dberror.Mapping{Entity: "PayrollExport"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workerports.TimesheetRepository {
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

func entriesQuery(req *workerports.ListTimeClockEntriesRequest) *keel.Query[worker.TimeClockEntry] {
	e := workerdb.TimeClockEntries
	q := e.Where(
		keel.When(req.WorkerID.IsNotNil(), e.WorkerID.Eq(req.WorkerID)),
		keel.When(req.TimesheetID.IsNotNil(), e.TimesheetID.Eq(req.TimesheetID)),
		keel.When(req.From > 0, e.ClockedInAt.Gte(req.From)),
		keel.When(req.To > 0, e.ClockedInAt.Lt(req.To)),
		keel.When(req.OpenOnly, e.ClockedOutAt.IsNull()),
		workerteam.Where(e.WorkerID.Col, req.ManagerIDs),
	).
		Order(e.ClockedInAt.Asc()).
		Limit(limitOr(req.Limit, defaultEntryPageSize))
	if req.IncludeWorker {
		q = q.Load(e.Worker().FleetCode())
	}

	return q
}

func (r *repository) ListEntries(
	ctx context.Context,
	req *workerports.ListTimeClockEntriesRequest,
) ([]*worker.TimeClockEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.TimeClockEntry, error) {
		entities, err := entriesQuery(req).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list time clock entries", zap.Error(err))
			return nil, fmt.Errorf("list time clock entries: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetEntryByID(
	ctx context.Context,
	req *workerports.GetTimeClockEntryByIDRequest,
) (*worker.TimeClockEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.TimeClockEntry, error) {
		entity, err := workerdb.TimeClockEntries.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, entryErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

// GetOpenEntry is the punch a worker is currently on. A missing one is not an
// error — most of the time nobody is on the clock — so it comes back nil.
func (r *repository) GetOpenEntry(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.TimeClockEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.TimeClockEntry, error) {
		e := workerdb.TimeClockEntries
		entity, err := e.Where(e.WorkerID.Eq(workerID), e.ClockedOutAt.IsNull()).
			Order(e.ClockedInAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // nobody on the clock is not an error
			}
			r.l.Error("failed to read open time clock entry", zap.Error(err))
			return nil, fmt.Errorf("read open time clock entry: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) CreateEntry(
	ctx context.Context,
	entity *worker.TimeClockEntry,
) (*worker.TimeClockEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.TimeClockEntry, error) {
		if err := workerdb.TimeClockEntries.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create time clock entry", zap.Error(err))
			return nil, entryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateEntry(
	ctx context.Context,
	entity *worker.TimeClockEntry,
) (*worker.TimeClockEntry, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.TimeClockEntry, error) {
		if err := workerdb.TimeClockEntries.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update time clock entry", zap.Error(err))
			return nil, entryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) DeleteEntry(
	ctx context.Context,
	req *workerports.GetTimeClockEntryByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := workerdb.TimeClockEntries.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&worker.TimeClockEntry{ID: req.ID},
		)
		if err != nil {
			r.l.Error("failed to delete time clock entry", zap.Error(err))
			return entryErrors.Map(err, req.ID.String())
		}

		return nil
	})
}

func timesheetsQuery(req *workerports.ListTimesheetsRequest) *keel.Query[worker.Timesheet] {
	t := workerdb.Timesheets
	q := t.Where(
		keel.When(req.WorkerID.IsNotNil(), t.WorkerID.Eq(req.WorkerID)),
		keel.When(len(req.Statuses) > 0, t.Status.In(req.Statuses...)),
		keel.When(req.PeriodStart > 0, t.PeriodStart.Eq(req.PeriodStart)),
		keel.When(req.From > 0, t.PeriodStart.Gte(req.From)),
		keel.When(req.To > 0, t.PeriodStart.Lt(req.To)),
		keel.When(req.ExportID.IsNotNil(), t.PayrollExportID.Eq(req.ExportID)),
		keel.When(req.UnexportedOnly, t.PayrollExportID.IsNull()),
		workerteam.Where(t.WorkerID.Col, req.ManagerIDs),
	).
		Order(t.PeriodStart.Desc()).
		Limit(limitOr(req.Limit, defaultTimesheetPageSize))
	if req.IncludeWorker {
		q = q.Load(t.Worker())
	}

	return q
}

func (r *repository) ListTimesheets(
	ctx context.Context,
	req *workerports.ListTimesheetsRequest,
) ([]*worker.Timesheet, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.Timesheet, error) {
		entities, err := timesheetsQuery(req).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list timesheets", zap.Error(err))
			return nil, fmt.Errorf("list timesheets: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetTimesheetByID(
	ctx context.Context,
	req *workerports.GetTimesheetByIDRequest,
) (*worker.Timesheet, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.Timesheet, error) {
		t := workerdb.Timesheets
		q := t.Where(t.ID.Eq(req.ID))
		if req.IncludeWorker {
			q = q.Load(t.Worker())
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, timesheetErrors.Map(err, req.ID.String())
		}

		if req.IncludeEntries {
			entries, err := r.ListEntries(ctx, &workerports.ListTimeClockEntriesRequest{
				TenantInfo:  req.TenantInfo,
				TimesheetID: entity.ID,
			})
			if err != nil {
				return nil, err
			}
			entity.Entries = entries
		}

		return entity, nil
	})
}

// GetOrCreateTimesheet is the week a punch belongs to. Insert-on-conflict
// rather than read-then-write: two punches landing at once would otherwise both
// find no sheet and both try to make one.
func (r *repository) GetOrCreateTimesheet(
	ctx context.Context,
	entity *worker.Timesheet,
) (*worker.Timesheet, error) {
	t := workerdb.Timesheets

	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*worker.Timesheet]{
		Find: func(ctx context.Context) (*worker.Timesheet, error) {
			return t.Where(t.WorkerID.Eq(entity.WorkerID), t.PeriodStart.Eq(entity.PeriodStart)).
				One(ctx, r.db.Keel(ctx))
		},
		Create: func(ctx context.Context) error {
			_, err := t.CreateOrSkip(
				ctx,
				r.db.Keel(ctx),
				[]*worker.Timesheet{entity},
				t.OrganizationID,
				t.BusinessUnitID,
				t.WorkerID,
				t.PeriodStart,
			)
			if err != nil {
				r.l.Error("failed to open timesheet", zap.Error(err))
				return fmt.Errorf("open timesheet: %w", err)
			}

			return nil
		},
		Default: func() *worker.Timesheet { return entity },
		MapError: func(err error) error {
			return timesheetErrors.Map(err, entity.ID.String())
		},
	})
}

func (r *repository) UpdateTimesheet(
	ctx context.Context,
	entity *worker.Timesheet,
) (*worker.Timesheet, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.Timesheet, error) {
		if err := workerdb.Timesheets.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update timesheet", zap.Error(err))
			return nil, timesheetErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

// AttachEntries rolls a week's loose punches onto the sheet they belong to. It
// takes only closed entries: a punch nobody has finished is not part of any
// week yet, and pulling it in would freeze a total that is still moving.
func (r *repository) AttachEntries(
	ctx context.Context,
	_ pagination.TenantInfo,
	timesheetID pulid.ID,
	workerID pulid.ID,
	from, to int64,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		e := workerdb.TimeClockEntries
		affected, err := e.Where(
			e.WorkerID.Eq(workerID),
			e.ClockedInAt.Gte(from),
			e.ClockedInAt.Lt(to),
			e.ClockedOutAt.IsNotNull(),
			e.TimesheetID.IsNull(),
		).Update(ctx, r.db.Keel(ctx), e.TimesheetID.To(timesheetID), e.Version.Add(0))
		if err != nil {
			r.l.Error("failed to attach time clock entries", zap.Error(err))
			return 0, fmt.Errorf("attach time clock entries: %w", err)
		}

		return int(affected), nil
	})
}

func (r *repository) ListExports(
	ctx context.Context,
	req *workerports.ListPayrollExportsRequest,
) ([]*worker.PayrollExport, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.PayrollExport, error) {
		p := workerdb.PayrollExports
		entities, err := p.Query().
			Order(p.PeriodStart.Desc()).
			Limit(limitOr(req.Limit, defaultExportPageSize)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list payroll exports", zap.Error(err))
			return nil, fmt.Errorf("list payroll exports: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetExportByID(
	ctx context.Context,
	req *workerports.GetPayrollExportByIDRequest,
) (*worker.PayrollExport, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.PayrollExport, error) {
		entity, err := workerdb.PayrollExports.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, exportErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateExport(
	ctx context.Context,
	entity *worker.PayrollExport,
) (*worker.PayrollExport, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.PayrollExport, error) {
		if err := workerdb.PayrollExports.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create payroll export", zap.Error(err))
			return nil, exportErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateExport(
	ctx context.Context,
	entity *worker.PayrollExport,
) (*worker.PayrollExport, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.PayrollExport, error) {
		if err := workerdb.PayrollExports.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update payroll export", zap.Error(err))
			return nil, exportErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

// StampExport attaches a run to the sheets it carried and locks them in one
// statement. Locking them one at a time would leave a half-exported period if
// anything failed partway. The status guard is part of the statement rather
// than a prior read, so a sheet somebody reopened between the two is skipped
// instead of being locked out from under them.
func (r *repository) StampExport(
	ctx context.Context,
	req *workerports.StampExportRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if len(req.TimesheetIDs) == 0 {
			return 0, nil
		}

		t := workerdb.Timesheets
		affected, err := t.Where(
			t.ID.In(req.TimesheetIDs...),
			t.Status.Eq(worker.TimesheetApproved),
			t.PayrollExportID.IsNull(),
		).Update(ctx, r.db.Keel(ctx), t.PayrollExportID.To(req.ExportID), t.Status.To(req.Status))
		if err != nil {
			r.l.Error("failed to stamp payroll export", zap.Error(err))
			return 0, fmt.Errorf("stamp payroll export: %w", err)
		}

		return int(affected), nil
	})
}

// ApprovedLeaveRanges is the approved time off touching a window. Overlap
// rather than containment: a week off that starts on the Friday before still
// covers this week's Monday.
func (r *repository) ApprovedLeaveRanges(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
	from, to int64,
) ([]workerports.RotaRangeRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]workerports.RotaRangeRow, error) {
		p := workerdb.WorkerPTOs
		rows, err := keel.Into[workerports.RotaRangeRow](ctx, r.db.Keel(ctx), p.
			Pick(p.WorkerID, p.StartDate, p.EndDate).
			Where(
				p.WorkerID.Eq(workerID),
				p.Status.Eq(worker.PTOStatusApproved),
				p.StartDate.Lt(to),
				p.EndDate.Gte(from),
			))
		if err != nil {
			r.l.Error("failed to read approved leave", zap.Error(err))
			return nil, fmt.Errorf("read approved leave: %w", err)
		}

		return rows, nil
	})
}

// ClearExport detaches every sheet from a voided run and puts them back to
// Approved, which is the state they were in when the run picked them up.
func (r *repository) ClearExport(
	ctx context.Context,
	_ pagination.TenantInfo,
	exportID pulid.ID,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		t := workerdb.Timesheets
		affected, err := t.Where(t.PayrollExportID.Eq(exportID)).
			Update(ctx, r.db.Keel(ctx), t.PayrollExportID.ToNull(), t.Status.To(worker.TimesheetApproved))
		if err != nil {
			r.l.Error("failed to clear payroll export", zap.Error(err))
			return 0, fmt.Errorf("clear payroll export: %w", err)
		}

		return int(affected), nil
	})
}
