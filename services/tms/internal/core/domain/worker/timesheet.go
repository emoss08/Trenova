package worker

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	_ bun.BeforeAppendModelHook          = (*TimeClockEntry)(nil)
	_ validationframework.TenantedEntity = (*TimeClockEntry)(nil)
)

// TimeClockEntry is one period of work.
type TimeClockEntry struct {
	bun.BaseModel `bun:"table:time_clock_entries,alias:tce" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`
	TimesheetID    pulid.ID `json:"timesheetId"    bun:"timesheet_id,type:VARCHAR(100),nullzero"`

	Source TimeEntrySource `json:"source" bun:"source,type:time_entry_source_enum,notnull,default:'Clock'"`
	// ClockedOutAt of nil means the worker is still on the clock.
	ClockedInAt  int64    `json:"clockedInAt"  bun:"clocked_in_at,type:BIGINT,notnull"`
	ClockedOutAt *int64   `json:"clockedOutAt" bun:"clocked_out_at,type:BIGINT,nullzero"`
	BreakMinutes int32    `json:"breakMinutes" bun:"break_minutes,type:INTEGER,notnull"`
	PayCodeID    pulid.ID `json:"payCodeId"    bun:"pay_code_id,type:VARCHAR(100),nullzero"`
	Note         string   `json:"note"         bun:"note,type:VARCHAR(500),nullzero"`
	EditedByID   pulid.ID `json:"editedById"   bun:"edited_by_id,type:VARCHAR(100),nullzero"`
	EditReason   string   `json:"editReason"   bun:"edit_reason,type:VARCHAR(500),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker *Worker `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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
	minutes := int32((*e.ClockedOutAt - e.ClockedInAt) / secondsPerMinute) //nolint:gosec // bounded below
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

	span := int32((*e.ClockedOutAt - e.ClockedInAt) / secondsPerMinute) //nolint:gosec // checked above
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

func (e *TimeClockEntry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("tce_")
		}
		if e.Source == "" {
			e.Source = TimeEntrySourceClock
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
}

var (
	_ bun.BeforeAppendModelHook          = (*Timesheet)(nil)
	_ validationframework.TenantedEntity = (*Timesheet)(nil)
)

// Timesheet is one worker's week of paid hours.
//
// Its totals are frozen at submit rather than derived on read. Everywhere else
// a roll-up is derived, because a stored one goes stale; here the roll-up is
// the decision — what a manager approved and what payroll paid from — and
// recomputing it later would change a signed-off wage record.
type Timesheet struct {
	bun.BaseModel `bun:"table:timesheets,alias:tsh" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`

	Status      TimesheetStatus `json:"status"      bun:"status,type:timesheet_status_enum,notnull,default:'Open'"`
	PeriodStart int64           `json:"periodStart" bun:"period_start,type:BIGINT,notnull"`
	PeriodEnd   int64           `json:"periodEnd"   bun:"period_end,type:BIGINT,notnull"`

	RegularMinutes   int32 `json:"regularMinutes"   bun:"regular_minutes,type:INTEGER,notnull"`
	OvertimeMinutes  int32 `json:"overtimeMinutes"  bun:"overtime_minutes,type:INTEGER,notnull"`
	PaidLeaveMinutes int32 `json:"paidLeaveMinutes" bun:"paid_leave_minutes,type:INTEGER,notnull"`
	EntryCount       int32 `json:"entryCount"       bun:"entry_count,type:INTEGER,notnull"`
	// OvertimeThresholdMinutes is copied onto the sheet rather than read from a
	// setting, so changing the rule next quarter cannot restate a week that was
	// already approved.
	OvertimeThresholdMinutes int32 `json:"overtimeThresholdMinutes" bun:"overtime_threshold_minutes,type:INTEGER,notnull,default:2400"`

	SubmittedAt     *int64   `json:"submittedAt"     bun:"submitted_at,type:BIGINT,nullzero"`
	SubmittedByID   pulid.ID `json:"submittedById"   bun:"submitted_by_id,type:VARCHAR(100),nullzero"`
	ApprovedAt      *int64   `json:"approvedAt"      bun:"approved_at,type:BIGINT,nullzero"`
	ApprovedByID    pulid.ID `json:"approvedById"    bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	DecisionNote    string   `json:"decisionNote"    bun:"decision_note,type:VARCHAR(500),nullzero"`
	PayrollExportID pulid.ID `json:"payrollExportId" bun:"payroll_export_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker  *Worker           `json:"worker,omitempty"  bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Entries []*TimeClockEntry `json:"entries,omitempty" bun:"-"`
}

// TotalMinutes is everything the week is worth.
func (t *Timesheet) TotalMinutes() int32 {
	return t.RegularMinutes + t.OvertimeMinutes + t.PaidLeaveMinutes
}

// ApplyTotals splits worked minutes across the overtime threshold and records
// what the sheet adds up to. Paid leave sits outside the split: time nobody
// worked does not earn overtime, and counting it would pay a premium on a
// holiday somebody spent at home.
func (t *Timesheet) ApplyTotals(workedMinutes, paidLeaveMinutes int32, entryCount int32) {
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

func (t *Timesheet) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("tsh_")
		}
		if t.Status == "" {
			t.Status = TimesheetOpen
		}
		if t.OvertimeThresholdMinutes <= 0 {
			t.OvertimeThresholdMinutes = defaultOvertimeThreshold
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}

var (
	_ bun.BeforeAppendModelHook          = (*PayrollExport)(nil)
	_ validationframework.TenantedEntity = (*PayrollExport)(nil)
)

// PayrollExport is one run of approved timesheets handed to payroll. A sheet
// carries the export it went out on, so a period cannot be sent twice without
// somebody voiding the first run.
type PayrollExport struct {
	bun.BaseModel `bun:"table:payroll_exports,alias:pxb" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Status      PayrollExportStatus `json:"status"      bun:"status,type:payroll_export_status_enum,notnull,default:'Draft'"`
	PeriodStart int64               `json:"periodStart" bun:"period_start,type:BIGINT,notnull"`
	PeriodEnd   int64               `json:"periodEnd"   bun:"period_end,type:BIGINT,notnull"`

	TimesheetCount   int32 `json:"timesheetCount"   bun:"timesheet_count,type:INTEGER,notnull"`
	RegularMinutes   int32 `json:"regularMinutes"   bun:"regular_minutes,type:INTEGER,notnull"`
	OvertimeMinutes  int32 `json:"overtimeMinutes"  bun:"overtime_minutes,type:INTEGER,notnull"`
	PaidLeaveMinutes int32 `json:"paidLeaveMinutes" bun:"paid_leave_minutes,type:INTEGER,notnull"`

	GeneratedAt   *int64   `json:"generatedAt"   bun:"generated_at,type:BIGINT,nullzero"`
	GeneratedByID pulid.ID `json:"generatedById" bun:"generated_by_id,type:VARCHAR(100),nullzero"`
	VoidedAt      *int64   `json:"voidedAt"      bun:"voided_at,type:BIGINT,nullzero"`
	VoidReason    string   `json:"voidReason"    bun:"void_reason,type:VARCHAR(500),nullzero"`
	Note          string   `json:"note"          bun:"note,type:VARCHAR(500),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

func (p *PayrollExport) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("pxb_")
		}
		if p.Status == "" {
			p.Status = PayrollExportDraft
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
