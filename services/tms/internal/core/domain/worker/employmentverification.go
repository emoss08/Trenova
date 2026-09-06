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
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	ErrInvalidVerificationStatus = errors.New("invalid employment verification status")
	ErrInvalidVerificationMethod = errors.New("invalid employment verification method")
)

const (
	// SafetyHistoryLookbackYears is how far back the investigation reaches:
	// every DOT-regulated employer in the three years before the application
	// (49 CFR 391.23(a)(2)).
	SafetyHistoryLookbackYears = 3

	// SafetyHistoryDueDays is how long after hire the investigation must be
	// made (49 CFR 391.23(c)(1)).
	SafetyHistoryDueDays = 30
)

type EmploymentVerificationStatus string

const (
	VerificationPending       = EmploymentVerificationStatus("Pending")
	VerificationRequested     = EmploymentVerificationStatus("Requested")
	VerificationReceived      = EmploymentVerificationStatus("Received")
	VerificationNoResponse    = EmploymentVerificationStatus("NoResponse")
	VerificationNotApplicable = EmploymentVerificationStatus("NotApplicable")
)

func (s EmploymentVerificationStatus) String() string { return string(s) }

func (s EmploymentVerificationStatus) IsValid() bool {
	switch s {
	case VerificationPending, VerificationRequested, VerificationReceived,
		VerificationNoResponse, VerificationNotApplicable:
		return true
	default:
		return false
	}
}

// IsSettled reports whether this employer needs no further chasing. A previous
// employer who never answers still settles the obligation: the rule asks for a
// good-faith effort and a record of it, not for an answer nobody can compel.
func (s EmploymentVerificationStatus) IsSettled() bool {
	return s == VerificationReceived || s == VerificationNoResponse ||
		s == VerificationNotApplicable
}

func (s EmploymentVerificationStatus) Label() string {
	switch s {
	case VerificationPending:
		return "Not yet requested"
	case VerificationRequested:
		return "Awaiting response"
	case VerificationReceived:
		return "Response received"
	case VerificationNoResponse:
		return "No response after follow-up"
	case VerificationNotApplicable:
		return "Not applicable"
	default:
		return string(s)
	}
}

type EmploymentVerificationMethod string

const (
	VerificationByEmail  = EmploymentVerificationMethod("Email")
	VerificationByFax    = EmploymentVerificationMethod("Fax")
	VerificationByMail   = EmploymentVerificationMethod("Mail")
	VerificationByPhone  = EmploymentVerificationMethod("Phone")
	VerificationByPortal = EmploymentVerificationMethod("Portal")
	VerificationByOther  = EmploymentVerificationMethod("Other")
)

func (m EmploymentVerificationMethod) String() string { return string(m) }

func (m EmploymentVerificationMethod) IsValid() bool {
	switch m {
	case VerificationByEmail, VerificationByFax, VerificationByMail, VerificationByPhone,
		VerificationByPortal, VerificationByOther:
		return true
	default:
		return false
	}
}

var (
	_ bun.BeforeAppendModelHook          = (*WorkerEmploymentVerification)(nil)
	_ validationframework.TenantedEntity = (*WorkerEmploymentVerification)(nil)
)

// WorkerEmploymentVerification is one previous employer's safety performance
// history investigation (49 CFR 391.23). The drug and alcohol part is tracked
// separately because 49 CFR 382.413 asks for it specifically and an employer
// often answers the general request without it.
type WorkerEmploymentVerification struct {
	bun.BaseModel `bun:"table:worker_employment_verifications,alias:wemv" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`

	EmployerName      string `json:"employerName"      bun:"employer_name,type:VARCHAR(150),notnull"`
	EmployerDOTNumber string `json:"employerDotNumber" bun:"employer_dot_number,type:VARCHAR(20),nullzero"`
	EmployerMCNumber  string `json:"employerMcNumber"  bun:"employer_mc_number,type:VARCHAR(20),nullzero"`
	ContactName       string `json:"contactName"       bun:"contact_name,type:VARCHAR(100),nullzero"`
	ContactPhone      string `json:"contactPhone"      bun:"contact_phone,type:VARCHAR(30),nullzero"`
	ContactEmail      string `json:"contactEmail"      bun:"contact_email,type:VARCHAR(150),nullzero"`

	EmployedFrom    *int64 `json:"employedFrom"    bun:"employed_from,type:BIGINT,nullzero"`
	EmployedTo      *int64 `json:"employedTo"      bun:"employed_to,type:BIGINT,nullzero"`
	WasDOTRegulated bool   `json:"wasDotRegulated" bun:"was_dot_regulated,type:BOOLEAN,notnull"`

	Status EmploymentVerificationStatus `json:"status" bun:"status,type:employment_verification_status_enum,notnull,default:'Pending'"`
	Method EmploymentVerificationMethod `json:"method" bun:"method,type:employment_verification_method_enum,notnull,default:'Email'"`

	RequestedAt                   *int64 `json:"requestedAt"                   bun:"requested_at,type:BIGINT,nullzero"`
	ResponseReceivedAt            *int64 `json:"responseReceivedAt"            bun:"response_received_at,type:BIGINT,nullzero"`
	LastFollowUpAt                *int64 `json:"lastFollowUpAt"                bun:"last_follow_up_at,type:BIGINT,nullzero"`
	FollowUpCount                 int32  `json:"followUpCount"                 bun:"follow_up_count,type:INTEGER,notnull"`
	DrugAlcoholResponseReceivedAt *int64 `json:"drugAlcoholResponseReceivedAt" bun:"drug_alcohol_response_received_at,type:BIGINT,nullzero"`

	HadAccidents             bool  `json:"hadAccidents"             bun:"had_accidents,type:BOOLEAN,notnull"`
	AccidentCount            int32 `json:"accidentCount"            bun:"accident_count,type:INTEGER,notnull"`
	HadDrugAlcoholViolations bool  `json:"hadDrugAlcoholViolations" bun:"had_drug_alcohol_violations,type:BOOLEAN,notnull"`

	Findings      string   `json:"findings"      bun:"findings,type:TEXT,nullzero"`
	Notes         string   `json:"notes"         bun:"notes,type:TEXT,nullzero"`
	DocumentID    pulid.ID `json:"documentId"    bun:"document_id,type:VARCHAR(100),nullzero"`
	RequestedByID pulid.ID `json:"requestedById" bun:"requested_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker      *Worker            `json:"worker,omitempty"      bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document    *document.Document `json:"document,omitempty"    bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	RequestedBy *tenant.User       `json:"requestedBy,omitempty" bun:"rel:belongs-to,join:requested_by_id=id"`
}

func (v *WorkerEmploymentVerification) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(v,
		validation.Field(&v.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&v.EmployerName,
			validation.Required.Error("Employer name is required"),
			validation.Length(1, 150).Error("Employer name cannot exceed 150 characters"),
		),
		validation.Field(&v.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[EmploymentVerificationStatus]("Status is not valid"),
		),
		validation.Field(&v.Method,
			validation.Required.Error("Method is required"),
			domainvalidation.ValidEnum[EmploymentVerificationMethod]("Method is not valid"),
		),
		validation.Field(&v.EmployerDOTNumber,
			validation.Length(0, 20).Error("DOT number cannot exceed 20 characters"),
		),
		validation.Field(&v.ContactEmail,
			validation.Length(0, 150).Error("Contact email cannot exceed 150 characters"),
			is.EmailFormat.Error("Contact email is not a valid address"),
		),
	))

	if v.EmployedFrom != nil && v.EmployedTo != nil && *v.EmployedTo < *v.EmployedFrom {
		multiErr.Add(
			"employedTo",
			errortypes.ErrInvalid,
			"The end of the employment cannot pre-date its start",
		)
	}

	// A request that was never sent cannot be awaiting a response, and an
	// answer that never arrived cannot be recorded as received. Both would
	// leave a file that reads as investigated and is not.
	if v.Status != VerificationPending && v.Status != VerificationNotApplicable &&
		(v.RequestedAt == nil || *v.RequestedAt <= 0) {
		multiErr.Add(
			"requestedAt",
			errortypes.ErrRequired,
			"Record when the request was sent",
		)
	}
	if v.Status == VerificationReceived &&
		(v.ResponseReceivedAt == nil || *v.ResponseReceivedAt <= 0) {
		multiErr.Add(
			"responseReceivedAt",
			errortypes.ErrRequired,
			"Record when the response arrived",
		)
	}
	if v.ResponseReceivedAt != nil && v.RequestedAt != nil &&
		*v.ResponseReceivedAt < *v.RequestedAt {
		multiErr.Add(
			"responseReceivedAt",
			errortypes.ErrInvalid,
			"The response cannot pre-date the request",
		)
	}
	if v.HadAccidents && v.AccidentCount <= 0 {
		multiErr.Add(
			"accidentCount",
			errortypes.ErrRequired,
			"Record how many accidents the employer reported",
		)
	}
	if v.FollowUpCount < 0 {
		multiErr.Add("followUpCount", errortypes.ErrInvalid, "Follow-up count cannot be negative")
	}
}

func (v *WorkerEmploymentVerification) IsSettled() bool { return v.Status.IsSettled() }

// IsOutstanding reports whether this employer is still owed a request or a
// chase.
func (v *WorkerEmploymentVerification) IsOutstanding() bool { return !v.Status.IsSettled() }

// NeedsDrugAlcoholAnswer reports whether the 382.413 half is still missing.
// It only applies to DOT-regulated employment: a driver's time at a
// non-regulated employer has no testing record to ask about.
func (v *WorkerEmploymentVerification) NeedsDrugAlcoholAnswer() bool {
	if !v.WasDOTRegulated || v.Status == VerificationNotApplicable {
		return false
	}
	return v.DrugAlcoholResponseReceivedAt == nil || *v.DrugAlcoholResponseReceivedAt <= 0
}

// DueAt is when this investigation had to be complete: thirty days after the
// driver was hired (49 CFR 391.23(c)(1)).
func DueAtForHire(hireDate int64) int64 {
	if hireDate <= 0 {
		return 0
	}
	return hireDate + SafetyHistoryDueDays*secondsPerDay
}

func (v *WorkerEmploymentVerification) GetID() pulid.ID { return v.ID }

func (v *WorkerEmploymentVerification) GetCreatedAt() int64 { return v.CreatedAt }

func (v *WorkerEmploymentVerification) GetOrganizationID() pulid.ID { return v.OrganizationID }

func (v *WorkerEmploymentVerification) GetBusinessUnitID() pulid.ID { return v.BusinessUnitID }

func (v *WorkerEmploymentVerification) GetTableName() string {
	return "worker_employment_verifications"
}

func (v *WorkerEmploymentVerification) GetResourceType() string {
	return "worker_employment_verification"
}

func (v *WorkerEmploymentVerification) GetResourceID() string { return v.ID.String() }

func (v *WorkerEmploymentVerification) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if v.ID.IsNil() {
			v.ID = pulid.MustNew("wemv_")
		}
		if v.Status == "" {
			v.Status = VerificationPending
		}
		if v.Method == "" {
			v.Method = VerificationByEmail
		}
		v.CreatedAt = now
		v.UpdatedAt = now
	case *bun.UpdateQuery:
		v.UpdatedAt = now
	}

	return nil
}
