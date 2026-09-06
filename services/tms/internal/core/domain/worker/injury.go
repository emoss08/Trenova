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
	"github.com/uptrace/bun"
)

var (
	ErrInvalidCaseClassification = errors.New("invalid osha case classification")
	ErrInvalidIllnessType        = errors.New("invalid osha illness type")
	ErrInvalidInjuryTreatment    = errors.New("invalid injury treatment")
)

// MaxCountedDays is the cap 29 CFR 1904.7(b)(3)(viii) puts on the day counts:
// a case stops accruing at 180 days away or restricted, whichever it is.
const MaxCountedDays = int32(180)

// OSHACaseClassification is the column of the 300 log a case lands in. The
// order matters: the log records only the most serious outcome, so a case that
// was restricted and then went days-away is a days-away case.
type OSHACaseClassification string

const (
	CaseNotRecordable            = OSHACaseClassification("NotRecordable")
	CaseFirstAidOnly             = OSHACaseClassification("FirstAidOnly")
	CaseOtherRecordable          = OSHACaseClassification("OtherRecordable")
	CaseJobTransferOrRestriction = OSHACaseClassification("JobTransferOrRestriction")
	CaseDaysAway                 = OSHACaseClassification("DaysAway")
	CaseDeath                    = OSHACaseClassification("Death")
)

func (c OSHACaseClassification) String() string { return string(c) }

func (c OSHACaseClassification) IsValid() bool {
	switch c {
	case CaseNotRecordable, CaseFirstAidOnly, CaseOtherRecordable,
		CaseJobTransferOrRestriction, CaseDaysAway, CaseDeath:
		return true
	default:
		return false
	}
}

// IsRecordable reports whether the case belongs on the 300 log. First aid alone
// is explicitly not recordable (29 CFR 1904.7(b)(5)(ii)), which is the
// distinction the whole log turns on.
func (c OSHACaseClassification) IsRecordable() bool {
	switch c {
	case CaseOtherRecordable, CaseJobTransferOrRestriction, CaseDaysAway, CaseDeath:
		return true
	default:
		return false
	}
}

func (c OSHACaseClassification) Label() string {
	switch c {
	case CaseNotRecordable:
		return "Not recordable"
	case CaseFirstAidOnly:
		return "First aid only"
	case CaseOtherRecordable:
		return "Other recordable case"
	case CaseJobTransferOrRestriction:
		return "Job transfer or restriction"
	case CaseDaysAway:
		return "Days away from work"
	case CaseDeath:
		return "Death"
	default:
		return string(c)
	}
}

// OSHAIllnessType is the illness column of the 300A summary. Everything that is
// not an illness is an injury, which is the great majority of cases.
type OSHAIllnessType string

const (
	IllnessInjury               = OSHAIllnessType("Injury")
	IllnessSkinDisorder         = OSHAIllnessType("SkinDisorder")
	IllnessRespiratoryCondition = OSHAIllnessType("RespiratoryCondition")
	IllnessPoisoning            = OSHAIllnessType("Poisoning")
	IllnessHearingLoss          = OSHAIllnessType("HearingLoss")
	IllnessOther                = OSHAIllnessType("OtherIllness")
)

func (t OSHAIllnessType) String() string { return string(t) }

func (t OSHAIllnessType) IsValid() bool {
	switch t {
	case IllnessInjury, IllnessSkinDisorder, IllnessRespiratoryCondition, IllnessPoisoning,
		IllnessHearingLoss, IllnessOther:
		return true
	default:
		return false
	}
}

func (t OSHAIllnessType) Label() string {
	switch t {
	case IllnessInjury:
		return "Injury"
	case IllnessSkinDisorder:
		return "Skin disorder"
	case IllnessRespiratoryCondition:
		return "Respiratory condition"
	case IllnessPoisoning:
		return "Poisoning"
	case IllnessHearingLoss:
		return "Hearing loss"
	case IllnessOther:
		return "Other illness"
	default:
		return string(t)
	}
}

type InjuryTreatment string

const (
	TreatmentNone            = InjuryTreatment("None")
	TreatmentFirstAid        = InjuryTreatment("FirstAid")
	TreatmentMedical         = InjuryTreatment("MedicalTreatment")
	TreatmentEmergencyRoom   = InjuryTreatment("EmergencyRoom")
	TreatmentHospitalization = InjuryTreatment("Hospitalized")
)

func (t InjuryTreatment) String() string { return string(t) }

func (t InjuryTreatment) IsValid() bool {
	switch t {
	case TreatmentNone, TreatmentFirstAid, TreatmentMedical, TreatmentEmergencyRoom,
		TreatmentHospitalization:
		return true
	default:
		return false
	}
}

// BeyondFirstAid reports whether the treatment given is itself enough to make a
// case recordable. Medical treatment beyond first aid is one of the general
// recording criteria (29 CFR 1904.7(a)).
func (t InjuryTreatment) BeyondFirstAid() bool {
	return t == TreatmentMedical || t == TreatmentEmergencyRoom ||
		t == TreatmentHospitalization
}

type InjuryCaseStatus string

const (
	InjuryCaseOpen   = InjuryCaseStatus("Open")
	InjuryCaseClosed = InjuryCaseStatus("Closed")
)

func (s InjuryCaseStatus) String() string { return string(s) }

func (s InjuryCaseStatus) IsValid() bool {
	return s == InjuryCaseOpen || s == InjuryCaseClosed
}

type WorkersCompClaimStatus string

const (
	ClaimNotFiled = WorkersCompClaimStatus("NotFiled")
	ClaimFiled    = WorkersCompClaimStatus("Filed")
	ClaimAccepted = WorkersCompClaimStatus("Accepted")
	ClaimDenied   = WorkersCompClaimStatus("Denied")
	ClaimClosed   = WorkersCompClaimStatus("Closed")
)

func (s WorkersCompClaimStatus) String() string { return string(s) }

func (s WorkersCompClaimStatus) IsValid() bool {
	switch s {
	case ClaimNotFiled, ClaimFiled, ClaimAccepted, ClaimDenied, ClaimClosed:
		return true
	default:
		return false
	}
}

func (s WorkersCompClaimStatus) Label() string {
	switch s {
	case ClaimNotFiled:
		return "Not filed"
	case ClaimFiled:
		return "Filed"
	case ClaimAccepted:
		return "Accepted"
	case ClaimDenied:
		return "Denied"
	case ClaimClosed:
		return "Closed"
	default:
		return string(s)
	}
}

// SuggestClassification proposes the 300 log column from what was recorded.
// It is a suggestion the recorder can override, because recordability is a
// judgement the employer makes — but it should not be a judgement they have to
// make from a blank field.
func SuggestClassification(
	treatment InjuryTreatment,
	daysAway int32,
	daysRestricted int32,
) OSHACaseClassification {
	switch {
	case daysAway > 0:
		return CaseDaysAway
	case daysRestricted > 0:
		return CaseJobTransferOrRestriction
	case treatment.BeyondFirstAid():
		return CaseOtherRecordable
	case treatment == TreatmentFirstAid:
		return CaseFirstAidOnly
	default:
		return CaseNotRecordable
	}
}

var (
	_ bun.BeforeAppendModelHook          = (*WorkerInjury)(nil)
	_ validationframework.TenantedEntity = (*WorkerInjury)(nil)
)

// WorkerInjury is one injury or illness case. The recordable ones are the OSHA
// 300 log; the rest are kept because the decision not to record is itself worth
// a record.
type WorkerInjury struct {
	bun.BaseModel `bun:"table:worker_injuries,alias:winj" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`

	// CaseNumber restarts each calendar year, which is how the 300 log reads.
	CaseNumber int32 `json:"caseNumber" bun:"case_number,type:INTEGER,notnull"`
	CaseYear   int16 `json:"caseYear"   bun:"case_year,type:SMALLINT,notnull"`

	Classification OSHACaseClassification `json:"classification" bun:"classification,type:osha_case_classification_enum,notnull,default:'NotRecordable'"`
	IllnessType    OSHAIllnessType        `json:"illnessType"    bun:"illness_type,type:osha_illness_type_enum,notnull,default:'Injury'"`
	Treatment      InjuryTreatment        `json:"treatment"      bun:"treatment,type:injury_treatment_enum,notnull,default:'None'"`
	Status         InjuryCaseStatus       `json:"status"         bun:"status,type:injury_case_status_enum,notnull,default:'Open'"`

	OccurredAt       int64  `json:"occurredAt"       bun:"occurred_at,type:BIGINT,notnull"`
	ReportedAt       *int64 `json:"reportedAt"       bun:"reported_at,type:BIGINT,nullzero"`
	ReturnedToWorkAt *int64 `json:"returnedToWorkAt" bun:"returned_to_work_at,type:BIGINT,nullzero"`

	Location       string `json:"location"       bun:"location,type:VARCHAR(255),nullzero"`
	Description    string `json:"description"    bun:"description,type:TEXT,notnull"`
	BodyPart       string `json:"bodyPart"       bun:"body_part,type:VARCHAR(100),nullzero"`
	HarmfulAgent   string `json:"harmfulAgent"   bun:"harmful_agent,type:VARCHAR(255),nullzero"`
	DaysAway       int32  `json:"daysAway"       bun:"days_away,type:INTEGER,notnull"`
	DaysRestricted int32  `json:"daysRestricted" bun:"days_restricted,type:INTEGER,notnull"`
	PrivacyCase    bool   `json:"privacyCase"    bun:"privacy_case,type:BOOLEAN,notnull"`

	ClaimStatus   WorkersCompClaimStatus `json:"claimStatus"   bun:"claim_status,type:workers_comp_claim_status_enum,notnull,default:'NotFiled'"`
	ClaimNumber   string                 `json:"claimNumber"   bun:"claim_number,type:VARCHAR(100),nullzero"`
	ClaimCarrier  string                 `json:"claimCarrier"  bun:"claim_carrier,type:VARCHAR(150),nullzero"`
	ClaimFiledAt  *int64                 `json:"claimFiledAt"  bun:"claim_filed_at,type:BIGINT,nullzero"`
	ClaimClosedAt *int64                 `json:"claimClosedAt" bun:"claim_closed_at,type:BIGINT,nullzero"`

	SafetyEventID pulid.ID `json:"safetyEventId" bun:"safety_event_id,type:VARCHAR(100),nullzero"`
	DocumentID    pulid.ID `json:"documentId"    bun:"document_id,type:VARCHAR(100),nullzero"`
	Notes         string   `json:"notes"         bun:"notes,type:TEXT,nullzero"`
	RecordedByID  pulid.ID `json:"recordedById"  bun:"recorded_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker      *Worker            `json:"worker,omitempty"      bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	SafetyEvent *WorkerSafetyEvent `json:"safetyEvent,omitempty" bun:"rel:belongs-to,join:safety_event_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document    *document.Document `json:"document,omitempty"    bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	RecordedBy  *tenant.User       `json:"recordedBy,omitempty"  bun:"rel:belongs-to,join:recorded_by_id=id"`
}

func (i *WorkerInjury) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&i.Description,
			validation.Required.Error("Describe what happened"),
		),
		validation.Field(&i.OccurredAt, validation.Required.Error("Date of the injury is required")),
		validation.Field(&i.Classification,
			validation.Required.Error("Classification is required"),
			domainvalidation.ValidEnum[OSHACaseClassification]("Classification is not valid"),
		),
		validation.Field(&i.IllnessType,
			validation.Required.Error("Injury or illness type is required"),
			domainvalidation.ValidEnum[OSHAIllnessType]("Type is not valid"),
		),
		validation.Field(&i.Treatment,
			validation.Required.Error("Treatment is required"),
			domainvalidation.ValidEnum[InjuryTreatment]("Treatment is not valid"),
		),
		validation.Field(&i.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[InjuryCaseStatus]("Status is not valid"),
		),
		validation.Field(&i.ClaimStatus,
			validation.Required.Error("Claim status is required"),
			domainvalidation.ValidEnum[WorkersCompClaimStatus]("Claim status is not valid"),
		),
		validation.Field(&i.BodyPart,
			validation.Length(0, 100).Error("Body part cannot exceed 100 characters"),
		),
		validation.Field(&i.ClaimNumber,
			validation.Length(0, 100).Error("Claim number cannot exceed 100 characters"),
		),
	))

	i.validateDays(multiErr)
	i.validateClaim(multiErr)

	if i.ReturnedToWorkAt != nil && *i.ReturnedToWorkAt < i.OccurredAt {
		multiErr.Add(
			"returnedToWorkAt",
			errortypes.ErrInvalid,
			"The return to work cannot pre-date the injury",
		)
	}
	if i.ReportedAt != nil && *i.ReportedAt < i.OccurredAt {
		multiErr.Add(
			"reportedAt",
			errortypes.ErrInvalid,
			"The report cannot pre-date the injury",
		)
	}
}

func (i *WorkerInjury) validateDays(multiErr *errortypes.MultiError) {
	if i.DaysAway < 0 || i.DaysRestricted < 0 {
		multiErr.Add("daysAway", errortypes.ErrInvalid, "Day counts cannot be negative")
	}
	if i.DaysAway > MaxCountedDays || i.DaysRestricted > MaxCountedDays {
		multiErr.Add(
			"daysAway",
			errortypes.ErrInvalid,
			"A case stops counting at 180 days (29 CFR 1904.7(b)(3)(viii))",
		)
	}

	// The log records only the most serious outcome, so a case with days away
	// cannot be filed in a lesser column and disappear from the count that
	// matters.
	if i.DaysAway > 0 && i.Classification != CaseDaysAway && i.Classification != CaseDeath {
		multiErr.Add(
			"classification",
			errortypes.ErrInvalid,
			"A case with days away from work is a days-away case",
		)
	}
	if i.DaysRestricted > 0 && !i.Classification.IsRecordable() {
		multiErr.Add(
			"classification",
			errortypes.ErrInvalid,
			"A case with restricted days is recordable",
		)
	}
}

func (i *WorkerInjury) validateClaim(multiErr *errortypes.MultiError) {
	if i.ClaimStatus == ClaimNotFiled {
		return
	}
	if i.ClaimFiledAt == nil || *i.ClaimFiledAt <= 0 {
		multiErr.Add(
			"claimFiledAt",
			errortypes.ErrRequired,
			"Record when the claim was filed",
		)
	}
	if i.ClaimStatus == ClaimClosed && (i.ClaimClosedAt == nil || *i.ClaimClosedAt <= 0) {
		multiErr.Add(
			"claimClosedAt",
			errortypes.ErrRequired,
			"Record when the claim was closed",
		)
	}
}

func (i *WorkerInjury) IsRecordable() bool { return i.Classification.IsRecordable() }

func (i *WorkerInjury) IsOpen() bool { return i.Status == InjuryCaseOpen }

// LogName is the name that appears on the posted log. A privacy concern case
// (29 CFR 1904.29(b)(6)) shows the case number instead, and the real name lives
// only on the separate confidential list.
func (i *WorkerInjury) LogName() string {
	if i.PrivacyCase {
		return "Privacy Case"
	}
	if i.Worker == nil {
		return ""
	}
	return i.Worker.FirstName + " " + i.Worker.LastName
}

func (i *WorkerInjury) GetID() pulid.ID { return i.ID }

func (i *WorkerInjury) GetCreatedAt() int64 { return i.CreatedAt }

func (i *WorkerInjury) GetOrganizationID() pulid.ID { return i.OrganizationID }

func (i *WorkerInjury) GetBusinessUnitID() pulid.ID { return i.BusinessUnitID }

func (i *WorkerInjury) GetTableName() string { return "worker_injuries" }

func (i *WorkerInjury) GetResourceType() string { return "worker_injury" }

func (i *WorkerInjury) GetResourceID() string { return i.ID.String() }

func (i *WorkerInjury) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("winj_")
		}
		if i.Status == "" {
			i.Status = InjuryCaseOpen
		}
		if i.Classification == "" {
			i.Classification = CaseNotRecordable
		}
		if i.IllnessType == "" {
			i.IllnessType = IllnessInjury
		}
		if i.Treatment == "" {
			i.Treatment = TreatmentNone
		}
		if i.ClaimStatus == "" {
			i.ClaimStatus = ClaimNotFiled
		}
		i.CreatedAt = now
		i.UpdatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}
