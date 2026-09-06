package worker

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
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
	_ bun.BeforeAppendModelHook          = (*WorkerLeaveCase)(nil)
	_ validationframework.TenantedEntity = (*WorkerLeaveCase)(nil)
)

// WorkerLeaveCase is one qualifying reason for leave. Entitlement is drawn down
// by the entries under a case, so a case that has been open for months has used
// nothing until days are recorded against it.
type WorkerLeaveCase struct {
	bun.BaseModel `bun:"table:worker_leave_cases,alias:wlc" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`

	LeaveType LeaveType       `json:"leaveType" bun:"leave_type,type:worker_leave_type_enum,notnull,default:'FMLA'"`
	Status    LeaveCaseStatus `json:"status"    bun:"status,type:leave_case_status_enum,notnull,default:'Pending'"`
	Frequency LeaveFrequency  `json:"frequency" bun:"frequency,type:leave_frequency_enum,notnull,default:'Continuous'"`
	Reason    string          `json:"reason"    bun:"reason,type:VARCHAR(255),nullzero"`

	// FMLADesignated is the employer's decision that this leave counts against
	// the entitlement (29 CFR 825.301). Leave can qualify and still not be
	// designated, which is why it is its own flag rather than a leave type.
	FMLADesignated    bool `json:"fmlaDesignated"    bun:"fmla_designated,type:BOOLEAN,notnull"`
	MilitaryCaregiver bool `json:"militaryCaregiver" bun:"military_caregiver,type:BOOLEAN,notnull"`

	RequestedAt int64  `json:"requestedAt" bun:"requested_at,type:BIGINT,notnull"`
	StartsAt    int64  `json:"startsAt"    bun:"starts_at,type:BIGINT,notnull"`
	EndsAt      *int64 `json:"endsAt"      bun:"ends_at,type:BIGINT,nullzero"`
	DecidedAt   *int64 `json:"decidedAt"   bun:"decided_at,type:BIGINT,nullzero"`
	ClosedAt    *int64 `json:"closedAt"    bun:"closed_at,type:BIGINT,nullzero"`

	CertificationStatus      LeaveCertificationStatus `json:"certificationStatus"      bun:"certification_status,type:leave_certification_status_enum,notnull,default:'NotRequired'"`
	CertificationRequestedAt *int64                   `json:"certificationRequestedAt" bun:"certification_requested_at,type:BIGINT,nullzero"`
	CertificationDueAt       *int64                   `json:"certificationDueAt"       bun:"certification_due_at,type:BIGINT,nullzero"`
	CertificationReceivedAt  *int64                   `json:"certificationReceivedAt"  bun:"certification_received_at,type:BIGINT,nullzero"`
	RecertificationDueAt     *int64                   `json:"recertificationDueAt"     bun:"recertification_due_at,type:BIGINT,nullzero"`

	// EligibilityHoursWorked is the hours in the twelve months before the leave
	// began. The system has no timeclock, so the office records it; it is
	// optional because the tenure half of the test can be answered without it.
	EligibilityHoursWorked *int32 `json:"eligibilityHoursWorked" bun:"eligibility_hours_worked,type:INTEGER,nullzero"`

	EmploymentEventID pulid.ID `json:"employmentEventId" bun:"employment_event_id,type:VARCHAR(100),nullzero"`
	DocumentID        pulid.ID `json:"documentId"        bun:"document_id,type:VARCHAR(100),nullzero"`
	Notes             string   `json:"notes"             bun:"notes,type:TEXT,nullzero"`
	DecidedByID       pulid.ID `json:"decidedById"       bun:"decided_by_id,type:VARCHAR(100),nullzero"`
	RecordedByID      pulid.ID `json:"recordedById"      bun:"recorded_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker    *Worker             `json:"worker,omitempty"    bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document  *document.Document  `json:"document,omitempty"  bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	DecidedBy *tenant.User        `json:"decidedBy,omitempty" bun:"rel:belongs-to,join:decided_by_id=id"`
	Entries   []*WorkerLeaveEntry `json:"entries,omitempty"   bun:"rel:has-many,join:id=leave_case_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (c *WorkerLeaveCase) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("wlc_")
		}
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
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}

var (
	_ bun.BeforeAppendModelHook          = (*WorkerLeaveEntry)(nil)
	_ validationframework.TenantedEntity = (*WorkerLeaveEntry)(nil)
)

// WorkerLeaveEntry is one day of leave taken. Intermittent leave is many rows;
// continuous leave is a row per working day. Hours rather than days, because
// 29 CFR 825.205 lets intermittent leave be taken in the smallest increment the
// employer uses for any other absence.
type WorkerLeaveEntry struct {
	bun.BaseModel `bun:"table:worker_leave_entries,alias:wle" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`
	LeaveCaseID    pulid.ID `json:"leaveCaseId"    bun:"leave_case_id,type:VARCHAR(100),notnull"`

	UsedOn int64           `json:"usedOn" bun:"used_on,type:BIGINT,notnull"`
	Hours  decimal.Decimal `json:"hours"  bun:"hours,type:NUMERIC(6,2),notnull"`
	// CountsAgainstEntitlement is copied from the case when the day is
	// recorded. It lives on the entry so that undesignating a case later does
	// not silently rewrite what was already counted.
	CountsAgainstEntitlement bool `json:"countsAgainstEntitlement" bun:"counts_against_entitlement,type:BOOLEAN,notnull"`

	// PTOID is the paid time off this day was also taken as. FMLA runs
	// concurrently with paid leave, so a day is very often both.
	PTOID        pulid.ID `json:"ptoId"        bun:"pto_id,type:VARCHAR(100),nullzero"`
	Notes        string   `json:"notes"        bun:"notes,type:TEXT,nullzero"`
	RecordedByID pulid.ID `json:"recordedById" bun:"recorded_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	LeaveCase *WorkerLeaveCase `json:"leaveCase,omitempty" bun:"rel:belongs-to,join:leave_case_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (e *WorkerLeaveEntry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("wle_")
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
}
