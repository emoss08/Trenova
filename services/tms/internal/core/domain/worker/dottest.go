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
	ErrInvalidDOTTestType      = errors.New("invalid dot test type")
	ErrInvalidDOTTestSubstance = errors.New("invalid dot test substance")
	ErrInvalidDOTTestStatus    = errors.New("invalid dot test status")
	ErrInvalidDOTTestResult    = errors.New("invalid dot test result")
)

type DOTTestType string

const (
	DOTTestPreEmployment       = DOTTestType("PreEmployment")
	DOTTestRandom              = DOTTestType("Random")
	DOTTestPostAccident        = DOTTestType("PostAccident")
	DOTTestReasonableSuspicion = DOTTestType("ReasonableSuspicion")
	DOTTestReturnToDuty        = DOTTestType("ReturnToDuty")
	DOTTestFollowUp            = DOTTestType("FollowUp")
	DOTTestOther               = DOTTestType("Other")
)

func (t DOTTestType) String() string { return string(t) }

func (t DOTTestType) IsValid() bool {
	switch t {
	case DOTTestPreEmployment, DOTTestRandom, DOTTestPostAccident, DOTTestReasonableSuspicion,
		DOTTestReturnToDuty, DOTTestFollowUp, DOTTestOther:
		return true
	default:
		return false
	}
}

func (t DOTTestType) Label() string {
	switch t {
	case DOTTestPreEmployment:
		return "Pre-employment"
	case DOTTestRandom:
		return "Random"
	case DOTTestPostAccident:
		return "Post-accident"
	case DOTTestReasonableSuspicion:
		return "Reasonable suspicion"
	case DOTTestReturnToDuty:
		return "Return to duty"
	case DOTTestFollowUp:
		return "Follow-up"
	case DOTTestOther:
		return "Other"
	default:
		return string(t)
	}
}

// RequiresReason marks the types a regulator expects a narrative for: what the
// supervisor observed, or which accident triggered the collection.
func (t DOTTestType) RequiresReason() bool {
	return t == DOTTestReasonableSuspicion || t == DOTTestPostAccident
}

type DOTTestSubstance string

const (
	DOTSubstanceDrug    = DOTTestSubstance("Drug")
	DOTSubstanceAlcohol = DOTTestSubstance("Alcohol")
)

func (s DOTTestSubstance) String() string { return string(s) }

func (s DOTTestSubstance) IsValid() bool {
	return s == DOTSubstanceDrug || s == DOTSubstanceAlcohol
}

type DOTTestStatus string

const (
	DOTTestStatusScheduled      = DOTTestStatus("Scheduled")
	DOTTestStatusCollected      = DOTTestStatus("Collected")
	DOTTestStatusAwaitingResult = DOTTestStatus("AwaitingResult")
	DOTTestStatusCompleted      = DOTTestStatus("Completed")
	DOTTestStatusCancelled      = DOTTestStatus("Cancelled")
)

func (s DOTTestStatus) String() string { return string(s) }

func (s DOTTestStatus) IsValid() bool {
	switch s {
	case DOTTestStatusScheduled, DOTTestStatusCollected, DOTTestStatusAwaitingResult,
		DOTTestStatusCompleted, DOTTestStatusCancelled:
		return true
	default:
		return false
	}
}

// IsOpen reports whether the office is still waiting on this test. A scheduled
// collection that never happened is as open as a specimen sitting at the lab.
func (s DOTTestStatus) IsOpen() bool {
	return s == DOTTestStatusScheduled || s == DOTTestStatusCollected ||
		s == DOTTestStatusAwaitingResult
}

// CanTransitionTo keeps a test moving forward. A completed or cancelled test is
// terminal: a corrected result is a new test, so the original stays on file.
func (s DOTTestStatus) CanTransitionTo(next DOTTestStatus) bool {
	switch s {
	case DOTTestStatusScheduled:
		return next == DOTTestStatusCollected || next == DOTTestStatusCancelled
	case DOTTestStatusCollected:
		return next == DOTTestStatusAwaitingResult || next == DOTTestStatusCompleted ||
			next == DOTTestStatusCancelled
	case DOTTestStatusAwaitingResult:
		return next == DOTTestStatusCompleted || next == DOTTestStatusCancelled
	default:
		return false
	}
}

type DOTTestResult string

const (
	DOTResultPending        = DOTTestResult("Pending")
	DOTResultNegative       = DOTTestResult("Negative")
	DOTResultNegativeDilute = DOTTestResult("NegativeDilute")
	DOTResultPositive       = DOTTestResult("Positive")
	DOTResultRefusal        = DOTTestResult("Refusal")
	DOTResultAdulterated    = DOTTestResult("Adulterated")
	DOTResultSubstituted    = DOTTestResult("Substituted")
	DOTResultInvalid        = DOTTestResult("Invalid")
	DOTResultCancelled      = DOTTestResult("Cancelled")
)

func (r DOTTestResult) String() string { return string(r) }

func (r DOTTestResult) IsValid() bool {
	switch r {
	case DOTResultPending, DOTResultNegative, DOTResultNegativeDilute, DOTResultPositive,
		DOTResultRefusal, DOTResultAdulterated, DOTResultSubstituted, DOTResultInvalid,
		DOTResultCancelled:
		return true
	default:
		return false
	}
}

// IsViolation reports whether the result is one 49 CFR 382.501 treats as a
// prohibition: the driver comes off safety-sensitive duty until the
// return-to-duty process is finished. An adulterated or substituted specimen is
// a refusal, so it counts the same as a positive.
func (r DOTTestResult) IsViolation() bool {
	switch r {
	case DOTResultPositive, DOTResultRefusal, DOTResultAdulterated, DOTResultSubstituted:
		return true
	default:
		return false
	}
}

// IsNegative reports whether the result satisfies a testing requirement. A
// dilute negative is still a negative; whether to re-collect is the employer's
// policy call, not a prohibition.
func (r DOTTestResult) IsNegative() bool {
	return r == DOTResultNegative || r == DOTResultNegativeDilute
}

// AlcoholProhibitedThreshold is the concentration at or above which a driver is
// prohibited from performing safety-sensitive functions (49 CFR 382.201).
var AlcoholProhibitedThreshold = decimal.NewFromFloat(0.04)

// AlcoholStandDownThreshold is the concentration at or above which a driver
// must be taken off duty for 24 hours even though it is not a violation
// (49 CFR 382.505).
var AlcoholStandDownThreshold = decimal.NewFromFloat(0.02)

// ResultForConcentration grades an alcohol reading. Below the stand-down
// threshold the test passes; at or above the prohibition threshold it is a
// violation. Between the two the driver stands down but has not violated, so
// the result stays negative and the standing is unaffected.
func ResultForConcentration(concentration decimal.Decimal) DOTTestResult {
	if concentration.GreaterThanOrEqual(AlcoholProhibitedThreshold) {
		return DOTResultPositive
	}
	return DOTResultNegative
}

var (
	_ bun.BeforeAppendModelHook          = (*WorkerDOTTest)(nil)
	_ validationframework.TenantedEntity = (*WorkerDOTTest)(nil)
)

type WorkerDOTTest struct {
	bun.BaseModel `bun:"table:worker_dot_tests,alias:wdot" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`

	TestType  DOTTestType      `json:"testType"  bun:"test_type,type:dot_test_type_enum,notnull"`
	Substance DOTTestSubstance `json:"substance" bun:"substance,type:dot_test_substance_enum,notnull"`
	Status    DOTTestStatus    `json:"status"    bun:"status,type:dot_test_status_enum,notnull,default:'Scheduled'"`
	Result    DOTTestResult    `json:"result"    bun:"result,type:dot_test_result_enum,notnull,default:'Pending'"`
	IsDOT     bool             `json:"isDot"     bun:"is_dot,type:BOOLEAN,notnull"`
	Reason    string           `json:"reason"    bun:"reason,type:TEXT,nullzero"`

	ScheduledAt *int64 `json:"scheduledAt" bun:"scheduled_at,type:BIGINT,nullzero"`
	CollectedAt *int64 `json:"collectedAt" bun:"collected_at,type:BIGINT,nullzero"`
	ResultAt    *int64 `json:"resultAt"    bun:"result_at,type:BIGINT,nullzero"`

	CollectionSite string `json:"collectionSite" bun:"collection_site,type:VARCHAR(150),nullzero"`
	CollectorName  string `json:"collectorName"  bun:"collector_name,type:VARCHAR(100),nullzero"`
	SpecimenID     string `json:"specimenId"     bun:"specimen_id,type:VARCHAR(100),nullzero"`
	LabName        string `json:"labName"        bun:"lab_name,type:VARCHAR(100),nullzero"`
	MROName        string `json:"mroName"        bun:"mro_name,type:VARCHAR(100),nullzero"`
	MROVerifiedAt  *int64 `json:"mroVerifiedAt"  bun:"mro_verified_at,type:BIGINT,nullzero"`

	AlcoholConcentration *decimal.Decimal `json:"alcoholConcentration" bun:"alcohol_concentration,type:NUMERIC(4,3),nullzero"`

	SafetyEventID pulid.ID `json:"safetyEventId" bun:"safety_event_id,type:VARCHAR(100),nullzero"`
	DrawEntryID   pulid.ID `json:"drawEntryId"   bun:"draw_entry_id,type:VARCHAR(100),nullzero"`
	DocumentID    pulid.ID `json:"documentId"    bun:"document_id,type:VARCHAR(100),nullzero"`
	Notes         string   `json:"notes"         bun:"notes,type:TEXT,nullzero"`
	OrderedByID   pulid.ID `json:"orderedById"   bun:"ordered_by_id,type:VARCHAR(100),nullzero"`
	RecordedByID  pulid.ID `json:"recordedById"  bun:"recorded_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker      *Worker            `json:"worker,omitempty"      bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	SafetyEvent *WorkerSafetyEvent `json:"safetyEvent,omitempty" bun:"rel:belongs-to,join:safety_event_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document    *document.Document `json:"document,omitempty"    bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	OrderedBy   *tenant.User       `json:"orderedBy,omitempty"   bun:"rel:belongs-to,join:ordered_by_id=id"`
	RecordedBy  *tenant.User       `json:"recordedBy,omitempty"  bun:"rel:belongs-to,join:recorded_by_id=id"`
}

func (t *WorkerDOTTest) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&t.TestType,
			validation.Required.Error("Test type is required"),
			domainvalidation.ValidEnum[DOTTestType]("Test type is not valid"),
		),
		validation.Field(&t.Substance,
			validation.Required.Error("Substance is required"),
			domainvalidation.ValidEnum[DOTTestSubstance]("Substance must be Drug or Alcohol"),
		),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[DOTTestStatus]("Status is not valid"),
		),
		validation.Field(&t.Result,
			validation.Required.Error("Result is required"),
			domainvalidation.ValidEnum[DOTTestResult]("Result is not valid"),
		),
		validation.Field(&t.CollectionSite,
			validation.Length(0, 150).Error("Collection site cannot exceed 150 characters"),
		),
		validation.Field(&t.CollectorName,
			validation.Length(0, 100).Error("Collector cannot exceed 100 characters"),
		),
		validation.Field(&t.SpecimenID,
			validation.Length(0, 100).Error("Specimen ID cannot exceed 100 characters"),
		),
		validation.Field(&t.LabName,
			validation.Length(0, 100).Error("Laboratory cannot exceed 100 characters"),
		),
		validation.Field(&t.MROName,
			validation.Length(0, 100).Error("Medical review officer cannot exceed 100 characters"),
		),
	))

	if t.TestType.RequiresReason() && t.Reason == "" {
		multiErr.Add(
			"reason",
			errortypes.ErrRequired,
			"A {0} test must record what prompted it", t.TestType.Label(),
		)
	}

	t.validateSubstanceFields(multiErr)
	t.validateTimeline(multiErr)
}

// validateSubstanceFields keeps the two halves of a collection apart. Only a
// drug test passes through a medical review officer, and only an alcohol test
// produces a concentration; letting either cross over would put numbers on a
// record that cannot have produced them.
func (t *WorkerDOTTest) validateSubstanceFields(multiErr *errortypes.MultiError) {
	if t.Substance == DOTSubstanceAlcohol {
		if t.MROName != "" || t.MROVerifiedAt != nil {
			multiErr.Add(
				"mroName",
				errortypes.ErrInvalid,
				"An alcohol test is not reviewed by a medical review officer",
			)
		}
		return
	}

	if t.AlcoholConcentration != nil {
		multiErr.Add(
			"alcoholConcentration",
			errortypes.ErrInvalid,
			"A drug test does not produce an alcohol concentration",
		)
	}
}

func (t *WorkerDOTTest) validateTimeline(multiErr *errortypes.MultiError) {
	if t.AlcoholConcentration != nil {
		if t.AlcoholConcentration.IsNegative() || t.AlcoholConcentration.GreaterThan(decimal.NewFromInt(1)) {
			multiErr.Add(
				"alcoholConcentration",
				errortypes.ErrInvalid,
				"Alcohol concentration must be between 0.000 and 1.000",
			)
		}
	}

	if t.Status != DOTTestStatusScheduled && t.Status != DOTTestStatusCancelled &&
		(t.CollectedAt == nil || *t.CollectedAt <= 0) {
		multiErr.Add(
			"collectedAt",
			errortypes.ErrRequired,
			"A collection date is required once the specimen has been taken",
		)
	}

	if t.Status == DOTTestStatusCompleted {
		if t.Result == DOTResultPending {
			multiErr.Add("result", errortypes.ErrRequired, "A completed test must have a result")
		}
		if t.ResultAt == nil || *t.ResultAt <= 0 {
			multiErr.Add("resultAt", errortypes.ErrRequired, "A completed test must have a result date")
		}
	}

	if t.CollectedAt != nil && t.ResultAt != nil && *t.ResultAt < *t.CollectedAt {
		multiErr.Add("resultAt", errortypes.ErrInvalid, "The result cannot pre-date the collection")
	}
}

func (t *WorkerDOTTest) IsOpen() bool { return t.Status.IsOpen() }

// IsViolation reports whether this test stands as a prohibition. A cancelled
// test never counts, whatever the lab said before it was voided.
func (t *WorkerDOTTest) IsViolation() bool {
	return t.Status == DOTTestStatusCompleted && t.Result.IsViolation()
}

// IsPassed reports whether this test satisfied its requirement.
func (t *WorkerDOTTest) IsPassed() bool {
	return t.Status == DOTTestStatusCompleted && t.Result.IsNegative()
}

// EffectiveAt is the instant this test speaks for: the collection where there
// is one, otherwise the schedule.
func (t *WorkerDOTTest) EffectiveAt() int64 {
	if t.CollectedAt != nil && *t.CollectedAt > 0 {
		return *t.CollectedAt
	}
	if t.ScheduledAt != nil && *t.ScheduledAt > 0 {
		return *t.ScheduledAt
	}
	return t.CreatedAt
}

// ViolationTypeFor names the violation this test creates, so a refusal is not
// filed as a positive.
func (t *WorkerDOTTest) ViolationTypeFor() DOTViolationType {
	switch t.Result {
	case DOTResultRefusal, DOTResultAdulterated, DOTResultSubstituted:
		return DOTViolationTestRefusal
	case DOTResultPositive:
		if t.Substance == DOTSubstanceAlcohol {
			return DOTViolationAlcoholUse
		}
		return DOTViolationPositiveTest
	default:
		return DOTViolationOther
	}
}

func (t *WorkerDOTTest) GetID() pulid.ID { return t.ID }

func (t *WorkerDOTTest) GetCreatedAt() int64 { return t.CreatedAt }

func (t *WorkerDOTTest) GetOrganizationID() pulid.ID { return t.OrganizationID }

func (t *WorkerDOTTest) GetBusinessUnitID() pulid.ID { return t.BusinessUnitID }

func (t *WorkerDOTTest) GetTableName() string { return "worker_dot_tests" }

func (t *WorkerDOTTest) GetResourceType() string { return "worker_dot_test" }

func (t *WorkerDOTTest) GetResourceID() string { return t.ID.String() }

func (t *WorkerDOTTest) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("wdot_")
		}
		if t.Status == "" {
			t.Status = DOTTestStatusScheduled
		}
		if t.Result == "" {
			t.Result = DOTResultPending
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}
