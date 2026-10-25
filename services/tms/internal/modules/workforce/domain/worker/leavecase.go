package worker

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidLeaveCaseStatus     = errors.New("invalid leave case status")
	ErrInvalidLeaveFrequency      = errors.New("invalid leave frequency")
	ErrInvalidCertificationStatus = errors.New("invalid leave certification status")
)

type LeaveCaseStatus string

const (
	LeaveCasePending  = LeaveCaseStatus("Pending")
	LeaveCaseApproved = LeaveCaseStatus("Approved")
	LeaveCaseDenied   = LeaveCaseStatus("Denied")
	LeaveCaseClosed   = LeaveCaseStatus("Closed")
)

func (s LeaveCaseStatus) String() string { return string(s) }

func (s LeaveCaseStatus) IsValid() bool {
	switch s {
	case LeaveCasePending, LeaveCaseApproved, LeaveCaseDenied, LeaveCaseClosed:
		return true
	default:
		return false
	}
}

// IsOpen reports whether the case can still have leave taken against it.
func (s LeaveCaseStatus) IsOpen() bool {
	return s == LeaveCasePending || s == LeaveCaseApproved
}

func (s LeaveCaseStatus) Label() string {
	switch s {
	case LeaveCasePending:
		return "Awaiting a decision"
	case LeaveCaseApproved:
		return "Approved"
	case LeaveCaseDenied:
		return "Denied"
	case LeaveCaseClosed:
		return "Closed"
	default:
		return string(s)
	}
}

type LeaveFrequency string

const (
	LeaveContinuous      = LeaveFrequency("Continuous")
	LeaveIntermittent    = LeaveFrequency("Intermittent")
	LeaveReducedSchedule = LeaveFrequency("ReducedSchedule")
)

func (f LeaveFrequency) String() string { return string(f) }

func LeaveFrequencyValues() []LeaveFrequency {
	return []LeaveFrequency{
		LeaveContinuous,
		LeaveIntermittent,
		LeaveReducedSchedule,
	}
}

func (f LeaveFrequency) IsValid() bool {
	switch f {
	case LeaveContinuous, LeaveIntermittent, LeaveReducedSchedule:
		return true
	default:
		return false
	}
}

func (f LeaveFrequency) Label() string {
	switch f {
	case LeaveContinuous:
		return "Continuous"
	case LeaveIntermittent:
		return "Intermittent"
	case LeaveReducedSchedule:
		return "Reduced schedule"
	default:
		return string(f)
	}
}

type LeaveCertificationStatus string

const (
	CertificationNotRequired  = LeaveCertificationStatus("NotRequired")
	CertificationRequested    = LeaveCertificationStatus("Requested")
	CertificationReceived     = LeaveCertificationStatus("Received")
	CertificationInsufficient = LeaveCertificationStatus("Insufficient")
	CertificationOverdue      = LeaveCertificationStatus("Overdue")
	CertificationWaived       = LeaveCertificationStatus("Waived")
)

func (s LeaveCertificationStatus) String() string { return string(s) }

func (s LeaveCertificationStatus) IsValid() bool {
	switch s {
	case CertificationNotRequired, CertificationRequested, CertificationReceived,
		CertificationInsufficient, CertificationOverdue, CertificationWaived:
		return true
	default:
		return false
	}
}

// IsOutstanding reports whether the office is still waiting on paperwork the
// employee owes.
func (s LeaveCertificationStatus) IsOutstanding() bool {
	return s == CertificationRequested || s == CertificationInsufficient ||
		s == CertificationOverdue
}

func (s LeaveCertificationStatus) Label() string {
	switch s {
	case CertificationNotRequired:
		return "Not required"
	case CertificationRequested:
		return "Requested"
	case CertificationReceived:
		return "Received"
	case CertificationInsufficient:
		return "Incomplete or insufficient"
	case CertificationOverdue:
		return "Overdue"
	case CertificationWaived:
		return "Waived"
	default:
		return string(s)
	}
}

var (
	_ validationframework.TenantedEntity = (*WorkerLeaveCase)(nil)
)

// WorkerLeaveCase is one qualifying reason for leave. Entitlement is drawn down
// by the entries under a case, so a case that has been open for months has used
// nothing until days are recorded against it.
type WorkerLeaveCase struct {
	keel.Table `keel:"worker_leave_cases"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:wlc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID       pulid.ID `json:"workerId"       keel:"worker_id"`

	LeaveType LeaveType       `json:"leaveType" keel:"leave_type"`
	Status    LeaveCaseStatus `json:"status"    keel:"status"`
	Frequency LeaveFrequency  `json:"frequency" keel:"frequency"`
	Reason    string          `json:"reason"    keel:"reason,nullzero"`

	// FMLADesignated is the employer's decision that this leave counts against
	// the entitlement (29 CFR 825.301). Leave can qualify and still not be
	// designated, which is why it is its own flag rather than a leave type.
	FMLADesignated    bool `json:"fmlaDesignated"    keel:"fmla_designated"`
	MilitaryCaregiver bool `json:"militaryCaregiver" keel:"military_caregiver"`

	RequestedAt int64  `json:"requestedAt" keel:"requested_at"`
	StartsAt    int64  `json:"startsAt"    keel:"starts_at"`
	EndsAt      *int64 `json:"endsAt"      keel:"ends_at"`
	DecidedAt   *int64 `json:"decidedAt"   keel:"decided_at"`
	ClosedAt    *int64 `json:"closedAt"    keel:"closed_at"`

	CertificationStatus      LeaveCertificationStatus `json:"certificationStatus"      keel:"certification_status"`
	CertificationRequestedAt *int64                   `json:"certificationRequestedAt" keel:"certification_requested_at"`
	CertificationDueAt       *int64                   `json:"certificationDueAt"       keel:"certification_due_at"`
	CertificationReceivedAt  *int64                   `json:"certificationReceivedAt"  keel:"certification_received_at"`
	RecertificationDueAt     *int64                   `json:"recertificationDueAt"     keel:"recertification_due_at"`

	// EligibilityHoursWorked is the hours in the twelve months before the leave
	// began. The system has no timeclock, so the office records it; it is
	// optional because the tenure half of the test can be answered without it.
	EligibilityHoursWorked *int32 `json:"eligibilityHoursWorked" keel:"eligibility_hours_worked"`

	EmploymentEventID pulid.ID `json:"employmentEventId" keel:"employment_event_id,nullzero"`
	DocumentID        pulid.ID `json:"documentId"        keel:"document_id,nullzero"`
	Notes             string   `json:"notes"             keel:"notes,nullzero"`
	DecidedByID       pulid.ID `json:"decidedById"       keel:"decided_by_id,nullzero"`
	RecordedByID      pulid.ID `json:"recordedById"      keel:"recorded_by_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Worker    *Worker             `json:"worker,omitempty"    keel:"belongs:worker_id"`
	Document  *document.Document  `json:"document,omitempty"  keel:"belongs:document_id"`
	DecidedBy *tenant.User        `json:"decidedBy,omitempty" keel:"belongs:decided_by_id"`
	Entries   []*WorkerLeaveEntry `json:"entries,omitempty"   keel:"many:leave_case_id"`
}

func (c *WorkerLeaveCase) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&c.LeaveType,
			validation.Required.Error("Leave type is required"),
			domainvalidation.ValidEnum[LeaveType]("Leave type is not valid"),
		),
		validation.Field(&c.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[LeaveCaseStatus]("Status is not valid"),
		),
		validation.Field(&c.Frequency,
			validation.Required.Error("Frequency is required"),
			domainvalidation.ValidEnum[LeaveFrequency]("Frequency is not valid"),
		),
		validation.Field(&c.CertificationStatus,
			validation.Required.Error("Certification status is required"),
			domainvalidation.ValidEnum[LeaveCertificationStatus](
				"Certification status is not valid",
			),
		),
		validation.Field(&c.StartsAt, validation.Required.Error("Start date is required")),
		validation.Field(&c.Reason,
			validation.Length(0, 255).Error("Reason cannot exceed 255 characters"),
		),
	))

	if c.EndsAt != nil && *c.EndsAt < c.StartsAt {
		multiErr.Add("endsAt", errortypes.ErrInvalid, "Leave cannot end before it begins")
	}

	// A decision is a dated act. Recording one without the date leaves a case
	// that says it was approved and cannot say when.
	if c.Status != LeaveCasePending && (c.DecidedAt == nil || *c.DecidedAt <= 0) {
		multiErr.Add(
			"decidedAt",
			errortypes.ErrRequired,
			"Record when the decision was made",
		)
	}
	if c.Status == LeaveCaseClosed && (c.ClosedAt == nil || *c.ClosedAt <= 0) {
		multiErr.Add("closedAt", errortypes.ErrRequired, "Record when the case was closed")
	}

	c.validateCertification(multiErr)

	// Designating leave the employer has refused is a contradiction: a denied
	// case cannot draw down an entitlement.
	if c.FMLADesignated && c.Status == LeaveCaseDenied {
		multiErr.Add(
			"fmlaDesignated",
			errortypes.ErrInvalid,
			"A denied case cannot be designated as FMLA leave",
		)
	}
}

func (c *WorkerLeaveCase) validateCertification(multiErr *errortypes.MultiError) {
	if c.CertificationStatus.IsOutstanding() || c.CertificationStatus == CertificationReceived {
		if c.CertificationRequestedAt == nil || *c.CertificationRequestedAt <= 0 {
			multiErr.Add(
				"certificationRequestedAt",
				errortypes.ErrRequired,
				"Record when the certification was requested",
			)
		}
	}
	if c.CertificationStatus == CertificationReceived &&
		(c.CertificationReceivedAt == nil || *c.CertificationReceivedAt <= 0) {
		multiErr.Add(
			"certificationReceivedAt",
			errortypes.ErrRequired,
			"Record when the certification arrived",
		)
	}
	if c.CertificationReceivedAt != nil && c.CertificationRequestedAt != nil &&
		*c.CertificationReceivedAt < *c.CertificationRequestedAt {
		multiErr.Add(
			"certificationReceivedAt",
			errortypes.ErrInvalid,
			"The certification cannot arrive before it was asked for",
		)
	}
}

func (c *WorkerLeaveCase) IsOpen() bool { return c.Status.IsOpen() }

// CountsAgainstEntitlement reports whether days under this case draw the FMLA
// balance down. Only designated leave does; everything else is recorded but not
// counted.
func (c *WorkerLeaveCase) CountsAgainstEntitlement() bool {
	return c.FMLADesignated && c.Status != LeaveCaseDenied
}

// CertificationLate reports whether the employee has run past the deadline.
// It is derived rather than stored so a case does not need a sweep to become
// late — it simply is, once the date passes.
func (c *WorkerLeaveCase) CertificationLate(now int64) bool {
	if c.CertificationStatus != CertificationRequested {
		return false
	}
	return c.CertificationDueAt != nil && *c.CertificationDueAt > 0 && now > *c.CertificationDueAt
}

func (c *WorkerLeaveCase) GetID() pulid.ID { return c.ID }

func (c *WorkerLeaveCase) GetCreatedAt() int64 { return c.CreatedAt }

func (c *WorkerLeaveCase) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *WorkerLeaveCase) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *WorkerLeaveCase) GetTableName() string { return "worker_leave_cases" }

func (c *WorkerLeaveCase) GetResourceType() string { return "worker_leave_case" }

func (c *WorkerLeaveCase) GetResourceID() string { return c.ID.String() }

func (c *WorkerLeaveCase) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if c.Status == "" {
		c.Status = LeaveCasePending
	}
	if c.LeaveType == "" {
		c.LeaveType = LeaveTypeFMLA
	}
	if c.Frequency == "" {
		c.Frequency = LeaveContinuous
	}
	if c.CertificationStatus == "" {
		c.CertificationStatus = CertificationNotRequired
	}
	if c.RequestedAt == 0 {
		c.RequestedAt = now
	}

	return nil
}

var (
	_ validationframework.TenantedEntity = (*WorkerLeaveEntry)(nil)
)

// WorkerLeaveEntry is one day of leave taken. Intermittent leave is many rows;
// continuous leave is a row per working day. Hours rather than days, because
// 29 CFR 825.205 lets intermittent leave be taken in the smallest increment the
// employer uses for any other absence.
type WorkerLeaveEntry struct {
	keel.Table `keel:"worker_leave_entries"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:wle_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID       pulid.ID `json:"workerId"       keel:"worker_id"`
	LeaveCaseID    pulid.ID `json:"leaveCaseId"    keel:"leave_case_id"`

	UsedOn int64           `json:"usedOn" keel:"used_on"`
	Hours  decimal.Decimal `json:"hours"  keel:"hours"`
	// CountsAgainstEntitlement is copied from the case when the day is
	// recorded. It lives on the entry so that undesignating a case later does
	// not silently rewrite what was already counted.
	CountsAgainstEntitlement bool `json:"countsAgainstEntitlement" keel:"counts_against_entitlement"`

	// PTOID is the paid time off this day was also taken as. FMLA runs
	// concurrently with paid leave, so a day is very often both.
	PTOID        pulid.ID `json:"ptoId"        keel:"pto_id,nullzero"`
	Notes        string   `json:"notes"        keel:"notes,nullzero"`
	RecordedByID pulid.ID `json:"recordedById" keel:"recorded_by_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	LeaveCase *WorkerLeaveCase `json:"leaveCase,omitempty" keel:"belongs:leave_case_id"`
}

// MaxLeaveHoursPerDay is the ceiling on one day's entry. A day cannot hold more
// hours than it has.
var MaxLeaveHoursPerDay = decimal.NewFromInt(24)

func (e *WorkerLeaveEntry) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&e.LeaveCaseID, validation.Required.Error("Leave case is required")),
		validation.Field(&e.UsedOn, validation.Required.Error("Date is required")),
	))

	if !e.Hours.IsPositive() {
		multiErr.Add("hours", errortypes.ErrInvalid, "Record more than zero hours")
	}
	if e.Hours.GreaterThan(MaxLeaveHoursPerDay) {
		multiErr.Add("hours", errortypes.ErrInvalid, "A day cannot hold more than 24 hours")
	}
}

func (e *WorkerLeaveEntry) GetID() pulid.ID { return e.ID }

func (e *WorkerLeaveEntry) GetCreatedAt() int64 { return e.CreatedAt }

func (e *WorkerLeaveEntry) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *WorkerLeaveEntry) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *WorkerLeaveEntry) GetTableName() string { return "worker_leave_entries" }

func (e *WorkerLeaveEntry) GetResourceType() string { return "worker_leave_entry" }

func (e *WorkerLeaveEntry) GetResourceID() string { return e.ID.String() }
