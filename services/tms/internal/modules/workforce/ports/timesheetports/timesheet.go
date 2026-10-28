package timesheetports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerscheduleports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListTimeClockEntriesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	// TimesheetID narrows to the entries rolled into one week, which is what
	// the approval screen reads.
	TimesheetID pulid.ID `json:"timesheetId"`
	From        int64    `json:"from"`
	To          int64    `json:"to"`
	OpenOnly    bool     `json:"openOnly"`
	// ManagerIDs narrows the list to the people a manager answers for, the
	// same way the timesheet queue is narrowed.
	ManagerIDs    []pulid.ID `json:"managerIds"`
	IncludeWorker bool       `json:"includeWorker"`
	Limit         int        `json:"limit"`
}

type GetTimeClockEntryByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListTimesheetsRequest struct {
	TenantInfo pagination.TenantInfo       `json:"tenantInfo"`
	WorkerID   pulid.ID                    `json:"workerId"`
	Statuses   []timesheet.TimesheetStatus `json:"statuses"`
	// PeriodStart pins one week; From and To scan a range of them.
	PeriodStart int64 `json:"periodStart"`
	From        int64 `json:"from"`
	To          int64 `json:"to"`
	// ManagerIDs narrows the queue to the people a manager answers for.
	ManagerIDs     []pulid.ID `json:"managerIds"`
	IncludeWorker  bool       `json:"includeWorker"`
	ExportID       pulid.ID   `json:"exportId"`
	UnexportedOnly bool       `json:"unexportedOnly"`
	Limit          int        `json:"limit"`
}

type GetTimesheetByIDRequest struct {
	ID             pulid.ID              `json:"id"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	IncludeEntries bool                  `json:"includeEntries"`
	IncludeWorker  bool                  `json:"includeWorker"`
}

type ListPayrollExportsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Limit      int                   `json:"limit"`
}

type GetPayrollExportByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

// StampExportRequest attaches a payroll run to the sheets it carried and locks
// them in one statement. Locking them one at a time would leave a half-exported
// period if anything failed partway.
type StampExportRequest struct {
	TenantInfo   pagination.TenantInfo
	ExportID     pulid.ID
	TimesheetIDs []pulid.ID
	Status       timesheet.TimesheetStatus
}

type TimesheetRepository interface {
	ListEntries(
		ctx context.Context,
		req *ListTimeClockEntriesRequest,
	) ([]*timesheet.TimeClockEntry, error)
	GetEntryByID(
		ctx context.Context,
		req *GetTimeClockEntryByIDRequest,
	) (*timesheet.TimeClockEntry, error)
	// GetOpenEntry is the punch a worker is currently on, if any. It is a
	// query of its own rather than a filter because every clock action asks it.
	GetOpenEntry(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) (*timesheet.TimeClockEntry, error)
	CreateEntry(
		ctx context.Context,
		entity *timesheet.TimeClockEntry,
	) (*timesheet.TimeClockEntry, error)
	UpdateEntry(
		ctx context.Context,
		entity *timesheet.TimeClockEntry,
	) (*timesheet.TimeClockEntry, error)
	DeleteEntry(ctx context.Context, req *GetTimeClockEntryByIDRequest) error

	ListTimesheets(ctx context.Context, req *ListTimesheetsRequest) ([]*timesheet.Timesheet, error)
	GetTimesheetByID(
		ctx context.Context,
		req *GetTimesheetByIDRequest,
	) (*timesheet.Timesheet, error)
	// GetOrCreateTimesheet is the week a punch belongs to. Insert-on-conflict
	// rather than read-then-write: two punches landing at once would otherwise
	// both find no sheet and both try to make one.
	GetOrCreateTimesheet(
		ctx context.Context,
		entity *timesheet.Timesheet,
	) (*timesheet.Timesheet, error)
	UpdateTimesheet(ctx context.Context, entity *timesheet.Timesheet) (*timesheet.Timesheet, error)
	// AttachEntries rolls a week's loose punches onto the sheet they belong to.
	AttachEntries(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		timesheetID pulid.ID,
		workerID pulid.ID,
		from, to int64,
	) (int, error)

	ListExports(
		ctx context.Context,
		req *ListPayrollExportsRequest,
	) ([]*timesheet.PayrollExport, error)
	GetExportByID(
		ctx context.Context,
		req *GetPayrollExportByIDRequest,
	) (*timesheet.PayrollExport, error)
	CreateExport(
		ctx context.Context,
		entity *timesheet.PayrollExport,
	) (*timesheet.PayrollExport, error)
	UpdateExport(
		ctx context.Context,
		entity *timesheet.PayrollExport,
	) (*timesheet.PayrollExport, error)
	StampExport(ctx context.Context, req *StampExportRequest) (int, error)
	// ApprovedLeaveRanges is the approved time off overlapping a window, as the
	// spans it is stored as. It is read here rather than through the time-off
	// repository because that one answers "what is coming up" - spans fully
	// inside a window - and a week's paid leave is anything that touches it.
	ApprovedLeaveRanges(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		from, to int64,
	) ([]workerscheduleports.RotaRangeRow, error)
	// ClearExport detaches every sheet from a voided run and reopens them.
	ClearExport(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		exportID pulid.ID,
	) (int, error)
}
