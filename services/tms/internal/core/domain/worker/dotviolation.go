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
	ErrInvalidDOTViolationType   = errors.New("invalid dot violation type")
	ErrInvalidDOTViolationStatus = errors.New("invalid dot violation status")
)

// MinimumFollowUpTests is the floor a substance abuse professional may set for
// the follow-up programme: at least six unannounced tests in the first twelve
// months back on duty (49 CFR 382.311).
const MinimumFollowUpTests = 6

type DOTViolationType string

const (
	DOTViolationPositiveTest    = DOTViolationType("PositiveTest")
	DOTViolationTestRefusal     = DOTViolationType("TestRefusal")
	DOTViolationAlcoholUse      = DOTViolationType("AlcoholUse")
	DOTViolationDrugUse         = DOTViolationType("DrugUse")
	DOTViolationActualKnowledge = DOTViolationType("ActualKnowledge")
	DOTViolationOther           = DOTViolationType("Other")
)

func (t DOTViolationType) String() string { return string(t) }

func (t DOTViolationType) IsValid() bool {
	switch t {
	case DOTViolationPositiveTest, DOTViolationTestRefusal, DOTViolationAlcoholUse,
		DOTViolationDrugUse, DOTViolationActualKnowledge, DOTViolationOther:
		return true
	default:
		return false
	}
}

func (t DOTViolationType) Label() string {
	switch t {
	case DOTViolationPositiveTest:
		return "Positive test"
	case DOTViolationTestRefusal:
		return "Refusal to test"
	case DOTViolationAlcoholUse:
		return "Alcohol use"
	case DOTViolationDrugUse:
		return "Drug use"
	case DOTViolationActualKnowledge:
		return "Actual knowledge"
	case DOTViolationOther:
		return "Other"
	default:
		return string(t)
	}
}

// DOTViolationStatus is where the return-to-duty process has got to. The order
// is the order the regulation requires: referral, evaluation, a passed
// return-to-duty test, then the follow-up programme.
type DOTViolationStatus string

const (
	DOTViolationStatusOpen          = DOTViolationStatus("Open")
	DOTViolationStatusSAPEvaluation = DOTViolationStatus("SAPEvaluation")
	DOTViolationStatusRTDPending    = DOTViolationStatus("RTDPending")
	DOTViolationStatusFollowUp      = DOTViolationStatus("FollowUp")
	DOTViolationStatusResolved      = DOTViolationStatus("Resolved")
)

func (s DOTViolationStatus) String() string { return string(s) }

func (s DOTViolationStatus) IsValid() bool {
	switch s {
	case DOTViolationStatusOpen, DOTViolationStatusSAPEvaluation, DOTViolationStatusRTDPending,
		DOTViolationStatusFollowUp, DOTViolationStatusResolved:
		return true
	default:
		return false
	}
}

// Prohibits reports whether a driver at this stage must be kept off
// safety-sensitive duty. Follow-up testing happens after the driver is back at
// work, so it is not a prohibition — the tests are unannounced, not a bar.
func (s DOTViolationStatus) Prohibits() bool {
	switch s {
	case DOTViolationStatusOpen, DOTViolationStatusSAPEvaluation, DOTViolationStatusRTDPending:
		return true
	default:
		return false
	}
}

func (s DOTViolationStatus) Label() string {
	switch s {
	case DOTViolationStatusOpen:
		return "Awaiting SAP referral"
	case DOTViolationStatusSAPEvaluation:
		return "SAP evaluation"
	case DOTViolationStatusRTDPending:
		return "Return-to-duty test required"
	case DOTViolationStatusFollowUp:
		return "Follow-up testing"
	case DOTViolationStatusResolved:
		return "Resolved"
	default:
		return string(s)
	}
}

// ReturnToDutyStatus is the worker-level projection of the process, kept on the
// profile so the roster can be filtered without reading every violation.
type ReturnToDutyStatus string

const (
	ReturnToDutyNotRequired     = ReturnToDutyStatus("NotRequired")
	ReturnToDutySAPEvaluation   = ReturnToDutyStatus("SAPEvaluation")
	ReturnToDutyRTDTestRequired = ReturnToDutyStatus("RTDTestRequired")
	ReturnToDutyFollowUpTesting = ReturnToDutyStatus("FollowUpTesting")
	ReturnToDutyComplete        = ReturnToDutyStatus("Complete")
)

func (s ReturnToDutyStatus) String() string { return string(s) }

func (s ReturnToDutyStatus) IsValid() bool {
	switch s {
	case ReturnToDutyNotRequired, ReturnToDutySAPEvaluation, ReturnToDutyRTDTestRequired,
		ReturnToDutyFollowUpTesting, ReturnToDutyComplete:
		return true
	default:
		return false
	}
}

// ReturnToDutyStatusFor projects a violation's stage onto the worker.
func ReturnToDutyStatusFor(status DOTViolationStatus) ReturnToDutyStatus {
	switch status {
	case DOTViolationStatusOpen, DOTViolationStatusSAPEvaluation:
		return ReturnToDutySAPEvaluation
	case DOTViolationStatusRTDPending:
		return ReturnToDutyRTDTestRequired
	case DOTViolationStatusFollowUp:
		return ReturnToDutyFollowUpTesting
	case DOTViolationStatusResolved:
		return ReturnToDutyComplete
	default:
		return ReturnToDutyNotRequired
	}
}

var (
	_ bun.BeforeAppendModelHook          = (*WorkerDOTViolation)(nil)
	_ validationframework.TenantedEntity = (*WorkerDOTViolation)(nil)
)

type WorkerDOTViolation struct {
	bun.BaseModel `bun:"table:worker_dot_violations,alias:wdotv" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`

	ViolationType DOTViolationType   `json:"violationType" bun:"violation_type,type:dot_violation_type_enum,notnull"`
	Status        DOTViolationStatus `json:"status"        bun:"status,type:dot_violation_status_enum,notnull,default:'Open'"`
	OccurredAt    int64              `json:"occurredAt"    bun:"occurred_at,type:BIGINT,notnull"`

	SourceTestID              pulid.ID `json:"sourceTestId"              bun:"source_test_id,type:VARCHAR(100),nullzero"`
	ReportedToClearinghouseAt *int64   `json:"reportedToClearinghouseAt" bun:"reported_to_clearinghouse_at,type:BIGINT,nullzero"`

	SAPName                  string `json:"sapName"                  bun:"sap_name,type:VARCHAR(100),nullzero"`
	SAPReferredAt            *int64 `json:"sapReferredAt"            bun:"sap_referred_at,type:BIGINT,nullzero"`
	SAPEvaluationCompletedAt *int64 `json:"sapEvaluationCompletedAt" bun:"sap_evaluation_completed_at,type:BIGINT,nullzero"`

	RTDTestID      pulid.ID `json:"rtdTestId"      bun:"rtd_test_id,type:VARCHAR(100),nullzero"`
	RTDCompletedAt *int64   `json:"rtdCompletedAt" bun:"rtd_completed_at,type:BIGINT,nullzero"`

	FollowUpTestCount      int32  `json:"followUpTestCount"      bun:"follow_up_test_count,type:INTEGER,notnull"`
	FollowUpTestsCompleted int32  `json:"followUpTestsCompleted" bun:"follow_up_tests_completed,type:INTEGER,notnull"`
	FollowUpEndsAt         *int64 `json:"followUpEndsAt"         bun:"follow_up_ends_at,type:BIGINT,nullzero"`

	ResolvedAt   *int64   `json:"resolvedAt"   bun:"resolved_at,type:BIGINT,nullzero"`
	DocumentID   pulid.ID `json:"documentId"   bun:"document_id,type:VARCHAR(100),nullzero"`
	Notes        string   `json:"notes"        bun:"notes,type:TEXT,nullzero"`
	RecordedByID pulid.ID `json:"recordedById" bun:"recorded_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker     *Worker            `json:"worker,omitempty"     bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	SourceTest *WorkerDOTTest     `json:"sourceTest,omitempty" bun:"rel:belongs-to,join:source_test_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	RTDTest    *WorkerDOTTest     `json:"rtdTest,omitempty"    bun:"rel:belongs-to,join:rtd_test_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document   *document.Document `json:"document,omitempty"   bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	RecordedBy *tenant.User       `json:"recordedBy,omitempty" bun:"rel:belongs-to,join:recorded_by_id=id"`
}

func (v *WorkerDOTViolation) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(v,
		validation.Field(&v.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&v.ViolationType,
			validation.Required.Error("Violation type is required"),
			domainvalidation.ValidEnum[DOTViolationType]("Violation type is not valid"),
		),
		validation.Field(&v.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[DOTViolationStatus]("Status is not valid"),
		),
		validation.Field(&v.OccurredAt,
			validation.Required.Error("Violation date is required"),
		),
		validation.Field(&v.SAPName,
			validation.Length(0, 100).Error("SAP name cannot exceed 100 characters"),
		),
	))

	v.validateSequence(multiErr)

	if v.FollowUpTestCount < 0 {
		multiErr.Add(
			"followUpTestCount",
			errortypes.ErrInvalid,
			"Follow-up test count cannot be negative",
		)
	}
	if v.FollowUpTestsCompleted > v.FollowUpTestCount {
		multiErr.Add(
			"followUpTestsCompleted",
			errortypes.ErrInvalid,
			"More follow-up tests have been recorded than the programme calls for",
		)
	}
	if v.FollowUpTestCount > 0 && v.FollowUpTestCount < MinimumFollowUpTests {
		multiErr.Add(
			"followUpTestCount",
			errortypes.ErrInvalid,
			"A follow-up programme is at least six tests (49 CFR 382.311)",
		)
	}
}

// validateSequence keeps the return-to-duty steps in the order the regulation
// requires. Recording a passed return-to-duty test before the SAP has evaluated
// the driver would leave a file that reads as compliant and is not.
func (v *WorkerDOTViolation) validateSequence(multiErr *errortypes.MultiError) {
	if v.SAPEvaluationCompletedAt != nil && v.SAPReferredAt == nil {
		multiErr.Add(
			"sapReferredAt",
			errortypes.ErrRequired,
			"Record the SAP referral before the evaluation that followed it",
		)
	}
	if v.RTDCompletedAt != nil && v.SAPEvaluationCompletedAt == nil {
		multiErr.Add(
			"rtdCompletedAt",
			errortypes.ErrInvalid,
			"A return-to-duty test comes after the SAP evaluation (49 CFR 40.305)",
		)
	}
	if v.RTDCompletedAt != nil && *v.RTDCompletedAt < v.OccurredAt {
		multiErr.Add(
			"rtdCompletedAt",
			errortypes.ErrInvalid,
			"The return-to-duty test cannot pre-date the violation",
		)
	}
	if v.FollowUpTestsCompleted > 0 && v.RTDCompletedAt == nil {
		multiErr.Add(
			"followUpTestsCompleted",
			errortypes.ErrInvalid,
			"Follow-up testing starts once the driver has returned to duty",
		)
	}
}

// DeriveStatus reads the stage off what has actually been recorded, so the
// status can never drift from the file underneath it.
func (v *WorkerDOTViolation) DeriveStatus() DOTViolationStatus {
	switch {
	case v.SAPReferredAt == nil:
		return DOTViolationStatusOpen
	case v.SAPEvaluationCompletedAt == nil:
		return DOTViolationStatusSAPEvaluation
	case v.RTDCompletedAt == nil:
		return DOTViolationStatusRTDPending
	case v.FollowUpTestCount > 0 && v.FollowUpTestsCompleted < v.FollowUpTestCount:
		return DOTViolationStatusFollowUp
	case v.FollowUpTestCount == 0:
		return DOTViolationStatusFollowUp
	default:
		return DOTViolationStatusResolved
	}
}

// ApplyDerivedStatus moves the row to the stage its own fields imply and stamps
// or clears the resolution instant so the two can never disagree.
func (v *WorkerDOTViolation) ApplyDerivedStatus(now int64) {
	v.Status = v.DeriveStatus()
	if v.Status == DOTViolationStatusResolved {
		if v.ResolvedAt == nil {
			resolved := now
			v.ResolvedAt = &resolved
		}
		return
	}
	v.ResolvedAt = nil
}

func (v *WorkerDOTViolation) IsResolved() bool { return v.Status == DOTViolationStatusResolved }

func (v *WorkerDOTViolation) Prohibits() bool { return v.Status.Prohibits() }

func (v *WorkerDOTViolation) GetID() pulid.ID { return v.ID }

func (v *WorkerDOTViolation) GetCreatedAt() int64 { return v.CreatedAt }

func (v *WorkerDOTViolation) GetOrganizationID() pulid.ID { return v.OrganizationID }

func (v *WorkerDOTViolation) GetBusinessUnitID() pulid.ID { return v.BusinessUnitID }

func (v *WorkerDOTViolation) GetTableName() string { return "worker_dot_violations" }

func (v *WorkerDOTViolation) GetResourceType() string { return "worker_dot_violation" }

func (v *WorkerDOTViolation) GetResourceID() string { return v.ID.String() }

func (v *WorkerDOTViolation) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if v.ID.IsNil() {
			v.ID = pulid.MustNew("wdotv_")
		}
		if v.Status == "" {
			v.Status = DOTViolationStatusOpen
		}
		v.CreatedAt = now
		v.UpdatedAt = now
	case *bun.UpdateQuery:
		v.UpdatedAt = now
	}

	return nil
}
