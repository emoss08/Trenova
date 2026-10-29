package timesheet

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	ErrInvalidTimesheetStatus     = errors.New("invalid timesheet status")
	ErrInvalidTimeEntrySource     = errors.New("invalid time entry source")
	ErrInvalidPayrollExportStatus = errors.New("invalid payroll export status")
)

const (
	secondsPerMinute         = int64(60)
	defaultOvertimeThreshold = int32(2400)
	maxTimeEntryMinutes      = int32(24 * 60)
	maxTimesheetWeekMinutes  = int32(7 * 24 * 60)
)

// TimeEntrySource is where a punch came from. It is kept because a manual
// entry and a clock punch are different kinds of evidence at a wage-and-hour
// audit, and a screen that cannot tell them apart cannot say which is which.
type TimeEntrySource string

const (
	TimeEntrySourceClock  = TimeEntrySource("Clock")
	TimeEntrySourcePortal = TimeEntrySource("Portal")
	TimeEntrySourceManual = TimeEntrySource("Manual")
	TimeEntrySourceImport = TimeEntrySource("Import")
)

func (s TimeEntrySource) String() string { return string(s) }

func (s TimeEntrySource) IsValid() bool {
	switch s {
	case TimeEntrySourceClock, TimeEntrySourcePortal, TimeEntrySourceManual, TimeEntrySourceImport:
		return true
	default:
		return false
	}
}

// TimesheetStatus is how far a week has got.
type TimesheetStatus string

const (
	// TimesheetOpen is a week still being worked or corrected.
	TimesheetOpen = TimesheetStatus("Open")
	// TimesheetSubmitted is a week handed to a manager. Its totals are frozen
	// from here on: they are what the manager is being asked to approve.
	TimesheetSubmitted = TimesheetStatus("Submitted")
	TimesheetApproved  = TimesheetStatus("Approved")
	// TimesheetRejected goes back to Open so the entries can be corrected.
	TimesheetRejected = TimesheetStatus("Rejected")
	// TimesheetLocked is a week that has gone to payroll. Nothing about it
	// changes again unless the export is voided.
	TimesheetLocked = TimesheetStatus("Locked")
)

func (s TimesheetStatus) String() string { return string(s) }

func (s TimesheetStatus) IsValid() bool {
	switch s {
	case TimesheetOpen, TimesheetSubmitted, TimesheetApproved, TimesheetRejected, TimesheetLocked:
		return true
	default:
		return false
	}
}

// IsEditable reports whether the entries behind the sheet can still change. A
// submitted sheet is frozen because its totals are what somebody is being
// asked to sign; an approved one because they already did.
func (s TimesheetStatus) IsEditable() bool {
	return s == TimesheetOpen || s == TimesheetRejected
}

// CanTransitionTo is the state machine. A rejected week returns to Open rather
// than to a dead end, because the point of rejecting it is to have it fixed.
func (s TimesheetStatus) CanTransitionTo(next TimesheetStatus) bool {
	switch s {
	case TimesheetOpen:
		return next == TimesheetSubmitted
	case TimesheetSubmitted:
		return next == TimesheetApproved || next == TimesheetRejected
	case TimesheetRejected:
		return next == TimesheetSubmitted
	case TimesheetApproved:
		// Locked when it goes to payroll. Back to Open is a manager taking an
		// approval back before the run, which stays possible because the
		// alternative is a wrong week going to payroll to keep a record tidy.
		return next == TimesheetLocked || next == TimesheetOpen
	case TimesheetLocked:
		// Only voiding the run brings a paid week back, and it comes back to
		// Approved, not Open: the hours were signed off and still are.
		return next == TimesheetApproved
	default:
		return false
	}
}

// PayrollExportStatus is how far a payroll run has got.
type PayrollExportStatus string

const (
	PayrollExportDraft     = PayrollExportStatus("Draft")
	PayrollExportGenerated = PayrollExportStatus("Generated")
	PayrollExportVoided    = PayrollExportStatus("Voided")
)

func (s PayrollExportStatus) String() string { return string(s) }

func (s PayrollExportStatus) IsValid() bool {
	switch s {
	case PayrollExportDraft, PayrollExportGenerated, PayrollExportVoided:
		return true
	default:
		return false
	}
}

var (
	_ validationframework.TenantedEntity = (*TimeClockEntry)(nil)
)

// TimeClockEntry is one period of work.
type TimeClockEntry struct {
	keel.Table `keel:"time_clock_entries"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:tce_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID       pulid.ID `json:"workerId"       keel:"worker_id"`
	TimesheetID    pulid.ID `json:"timesheetId"    keel:"timesheet_id,nullzero"`

	Source TimeEntrySource `json:"source" keel:"source"`
	// ClockedOutAt of nil means the worker is still on the clock.
	ClockedInAt  int64    `json:"clockedInAt"  keel:"clocked_in_at"`
	ClockedOutAt *int64   `json:"clockedOutAt" keel:"clocked_out_at"`
	BreakMinutes int32    `json:"breakMinutes" keel:"break_minutes"`
	PayCodeID    pulid.ID `json:"payCodeId"    keel:"pay_code_id,nullzero"`
	Note         string   `json:"note"         keel:"note,nullzero"`
	EditedByID   pulid.ID `json:"editedById"   keel:"edited_by_id,nullzero"`
	EditReason   string   `json:"editReason"   keel:"edit_reason,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Worker *worker.Worker `json:"worker,omitempty" keel:"belongs:worker_id"`
}

// IsOpen reports whether the worker is still on the clock.
func (e *TimeClockEntry) IsOpen() bool { return e.ClockedOutAt == nil || *e.ClockedOutAt <= 0 }

// PaidMinutes is what the entry is worth once the unpaid break is taken off.
// An open entry is worth nothing yet: paying for a shift somebody has not
// finished would make every roll-up depend on when it was read.
func (e *TimeClockEntry) PaidMinutes() int32 {
	if e.IsOpen() {
		return 0
	}
	minutes := intutils.SafeToInt32((*e.ClockedOutAt - e.ClockedInAt) / secondsPerMinute)
	minutes -= e.BreakMinutes
	if minutes < 0 {
		return 0
	}
	return minutes
}

func (e *TimeClockEntry) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&e.Source,
			validation.Required.Error("A source is required"),
			domainvalidation.ValidEnum[TimeEntrySource]("Source is not valid"),
		),
		validation.Field(&e.ClockedInAt, validation.Required.Error("A start time is required")),
		validation.Field(&e.Note,
			validation.Length(0, 500).Error("Note cannot exceed 500 characters"),
		),
	))

	if e.IsOpen() {
		// A break can only be taken out of a period that has ended: subtracting
		// it from a running entry would make the same shift shrink as it goes.
		if e.BreakMinutes > 0 {
			multiErr.Add(
				"breakMinutes",
				errortypes.ErrInvalid,
				"A break is recorded when the entry is closed",
			)
		}
		return
	}

	if *e.ClockedOutAt <= e.ClockedInAt {
		multiErr.Add("clockedOutAt", errortypes.ErrInvalid, "An entry cannot end before it began")
		return
	}

	span := intutils.SafeToInt32((*e.ClockedOutAt - e.ClockedInAt) / secondsPerMinute)
	// A punch nobody closed until the next day is a forgotten clock-out rather
	// than a day somebody worked straight through.
	if span > maxTimeEntryMinutes {
		multiErr.Add(
			"clockedOutAt",
			errortypes.ErrInvalid,
			"An entry cannot be longer than a day — correct the clock-out",
		)
	}
	if e.BreakMinutes < 0 {
		multiErr.Add("breakMinutes", errortypes.ErrInvalid, "A break cannot be negative")
	}
	if e.BreakMinutes >= span {
		multiErr.Add(
			"breakMinutes",
			errortypes.ErrInvalid,
			"The break is as long as the entry — nothing would be paid",
		)
	}
}

func (e *TimeClockEntry) GetID() pulid.ID { return e.ID }

func (e *TimeClockEntry) GetCreatedAt() int64 { return e.CreatedAt }

func (e *TimeClockEntry) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *TimeClockEntry) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *TimeClockEntry) GetTableName() string { return "time_clock_entries" }

func (e *TimeClockEntry) GetResourceType() string { return "time_clock_entry" }

func (e *TimeClockEntry) GetResourceID() string { return e.ID.String() }

func (e *TimeClockEntry) BeforeCreate(_ context.Context) error {
	if e.Source == "" {
		e.Source = TimeEntrySourceClock
	}

	return nil
}

func (TimeClockEntry) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_time_clock_entries_open",
			Message: "Worker is already on the clock",
		},
		{
			Name:    "fk_time_clock_entries_worker",
			Message: "Worker does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_time_clock_entries_timesheet",
			Message: "Timesheet does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_time_clock_entries_pay_code",
			Message: "Pay code does not exist or belongs to a different organization",
		},
	}
}

var (
	_ validationframework.TenantedEntity = (*Timesheet)(nil)
)

// Timesheet is one worker's week of paid hours.
//
// Its totals are frozen at submit rather than derived on read. Everywhere else
// a roll-up is derived, because a stored one goes stale; here the roll-up is
// the decision — what a manager approved and what payroll paid from — and
// recomputing it later would change a signed-off wage record.
type Timesheet struct {
	keel.Table `keel:"timesheets"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:tsh_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID       pulid.ID `json:"workerId"       keel:"worker_id"`

	Status      TimesheetStatus `json:"status"      keel:"status"`
	PeriodStart int64           `json:"periodStart" keel:"period_start"`
	PeriodEnd   int64           `json:"periodEnd"   keel:"period_end"`

	RegularMinutes   int32 `json:"regularMinutes"   keel:"regular_minutes"`
	OvertimeMinutes  int32 `json:"overtimeMinutes"  keel:"overtime_minutes"`
	PaidLeaveMinutes int32 `json:"paidLeaveMinutes" keel:"paid_leave_minutes"`
	EntryCount       int32 `json:"entryCount"       keel:"entry_count"`
	// OvertimeThresholdMinutes is copied onto the sheet rather than read from a
	// setting, so changing the rule next quarter cannot restate a week that was
	// already approved.
	OvertimeThresholdMinutes int32 `json:"overtimeThresholdMinutes" keel:"overtime_threshold_minutes"`

	SubmittedAt     *int64   `json:"submittedAt"     keel:"submitted_at"`
	SubmittedByID   pulid.ID `json:"submittedById"   keel:"submitted_by_id,nullzero"`
	ApprovedAt      *int64   `json:"approvedAt"      keel:"approved_at"`
	ApprovedByID    pulid.ID `json:"approvedById"    keel:"approved_by_id,nullzero"`
	DecisionNote    string   `json:"decisionNote"    keel:"decision_note,nullzero"`
	PayrollExportID pulid.ID `json:"payrollExportId" keel:"payroll_export_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Worker  *worker.Worker    `json:"worker,omitempty"  keel:"belongs:worker_id"`
	Entries []*TimeClockEntry `json:"entries,omitempty"`
}

// TotalMinutes is everything the week is worth.
func (t *Timesheet) TotalMinutes() int32 {
	return t.RegularMinutes + t.OvertimeMinutes + t.PaidLeaveMinutes
}

// ApplyTotals splits worked minutes across the overtime threshold and records
// what the sheet adds up to. Paid leave sits outside the split: time nobody
// worked does not earn overtime, and counting it would pay a premium on a
// holiday somebody spent at home.
func (t *Timesheet) ApplyTotals(workedMinutes, paidLeaveMinutes, entryCount int32) {
	threshold := t.OvertimeThresholdMinutes
	if threshold <= 0 {
		threshold = defaultOvertimeThreshold
	}

	if workedMinutes > threshold {
		t.RegularMinutes = threshold
		t.OvertimeMinutes = workedMinutes - threshold
	} else {
		t.RegularMinutes = workedMinutes
		t.OvertimeMinutes = 0
	}
	t.PaidLeaveMinutes = paidLeaveMinutes
	t.EntryCount = entryCount
}

func (t *Timesheet) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[TimesheetStatus]("Status is not valid"),
		),
		validation.Field(&t.PeriodStart, validation.Required.Error("A period start is required")),
		validation.Field(&t.DecisionNote,
			validation.Length(0, 500).Error("Note cannot exceed 500 characters"),
		),
	))

	if t.PeriodEnd <= t.PeriodStart {
		multiErr.Add("periodEnd", errortypes.ErrInvalid, "A period cannot end before it begins")
	}
	if t.OvertimeThresholdMinutes <= 0 {
		multiErr.Add(
			"overtimeThresholdMinutes",
			errortypes.ErrInvalid,
			"The overtime threshold has to be more than zero",
		)
	}
	if t.RegularMinutes < 0 || t.OvertimeMinutes < 0 || t.PaidLeaveMinutes < 0 {
		multiErr.Add("regularMinutes", errortypes.ErrInvalid, "Hours cannot be negative")
	}
	// More than a week of hours in a week is an entry somebody never closed,
	// and approving it would pay for it.
	if t.TotalMinutes() > maxTimesheetWeekMinutes {
		multiErr.Add(
			"regularMinutes",
			errortypes.ErrInvalid,
			"The week adds up to more hours than a week has — check for an entry that was never closed",
		)
	}
	// A decision is a dated act. One with no date leaves a sheet that says it
	// was approved and cannot say when.
	if t.Status == TimesheetApproved && (t.ApprovedAt == nil || *t.ApprovedAt <= 0) {
		multiErr.Add("approvedAt", errortypes.ErrRequired, "An approval needs the date it was made")
	}
	if t.Status == TimesheetSubmitted && (t.SubmittedAt == nil || *t.SubmittedAt <= 0) {
		multiErr.Add(
			"submittedAt",
			errortypes.ErrRequired,
			"A submission needs the date it was made",
		)
	}
}

func (t *Timesheet) GetID() pulid.ID { return t.ID }

func (t *Timesheet) GetCreatedAt() int64 { return t.CreatedAt }

func (t *Timesheet) GetOrganizationID() pulid.ID { return t.OrganizationID }

func (t *Timesheet) GetBusinessUnitID() pulid.ID { return t.BusinessUnitID }

func (t *Timesheet) GetTableName() string { return "timesheets" }

func (t *Timesheet) GetResourceType() string { return "timesheet" }

func (t *Timesheet) GetResourceID() string { return t.ID.String() }

func (t *Timesheet) BeforeCreate(_ context.Context) error {
	if t.Status == "" {
		t.Status = TimesheetOpen
	}
	if t.OvertimeThresholdMinutes <= 0 {
		t.OvertimeThresholdMinutes = defaultOvertimeThreshold
	}
	if t.OvertimeThresholdMinutes == 0 {
		t.OvertimeThresholdMinutes = 2400
	}

	return nil
}

func (Timesheet) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_timesheets_worker_period",
			Field:   "periodStart",
			Message: "Worker already has a timesheet for this period",
		},
		{
			Name:    "fk_timesheets_worker",
			Message: "Worker does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_timesheets_payroll_export",
			Message: "Payroll export does not exist or belongs to a different organization",
		},
	}
}

var (
	_ validationframework.TenantedEntity = (*PayrollExport)(nil)
)

// PayrollExport is one run of approved timesheets handed to payroll. A sheet
// carries the export it went out on, so a period cannot be sent twice without
// somebody voiding the first run.
type PayrollExport struct {
	keel.Table `keel:"payroll_exports"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:pxb_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Status      PayrollExportStatus `json:"status"      keel:"status"`
	PeriodStart int64               `json:"periodStart" keel:"period_start"`
	PeriodEnd   int64               `json:"periodEnd"   keel:"period_end"`

	TimesheetCount   int32 `json:"timesheetCount"   keel:"timesheet_count"`
	RegularMinutes   int32 `json:"regularMinutes"   keel:"regular_minutes"`
	OvertimeMinutes  int32 `json:"overtimeMinutes"  keel:"overtime_minutes"`
	PaidLeaveMinutes int32 `json:"paidLeaveMinutes" keel:"paid_leave_minutes"`

	GeneratedAt   *int64   `json:"generatedAt"   keel:"generated_at"`
	GeneratedByID pulid.ID `json:"generatedById" keel:"generated_by_id,nullzero"`
	VoidedAt      *int64   `json:"voidedAt"      keel:"voided_at"`
	VoidReason    string   `json:"voidReason"    keel:"void_reason,nullzero"`
	Note          string   `json:"note"          keel:"note,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (p *PayrollExport) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[PayrollExportStatus]("Status is not valid"),
		),
		validation.Field(&p.PeriodStart, validation.Required.Error("A period start is required")),
		validation.Field(&p.VoidReason,
			validation.Length(0, 500).Error("Reason cannot exceed 500 characters"),
		),
	))

	if p.PeriodEnd <= p.PeriodStart {
		multiErr.Add("periodEnd", errortypes.ErrInvalid, "A period cannot end before it begins")
	}
	// Voiding a payroll run reopens every week in it, so it is refused without
	// a reason: nobody can explain the reopening afterwards otherwise.
	if p.Status == PayrollExportVoided && p.VoidReason == "" {
		multiErr.Add("voidReason", errortypes.ErrRequired, "Voiding a run needs a reason")
	}
}

func (p *PayrollExport) GetID() pulid.ID { return p.ID }

func (p *PayrollExport) GetCreatedAt() int64 { return p.CreatedAt }

func (p *PayrollExport) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *PayrollExport) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *PayrollExport) GetTableName() string { return "payroll_exports" }

func (p *PayrollExport) GetResourceType() string { return "payroll_export" }

func (p *PayrollExport) GetResourceID() string { return p.ID.String() }

func (p *PayrollExport) BeforeCreate(_ context.Context) error {
	if p.Status == "" {
		p.Status = PayrollExportDraft
	}

	return nil
}
