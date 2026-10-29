package fleetsafety

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var ErrInvalidCSABasic = errors.New("invalid CSA BASIC")

// CSABasic is one of the seven Behavior Analysis and Safety Improvement
// Categories the FMCSA sorts roadside violations into. A safety director reads
// their fleet in these terms, so the roll-up speaks them rather than inventing
// its own grouping.
type CSABasic string

const (
	BasicUnsafeDriving        = CSABasic("UnsafeDriving")
	BasicHOSCompliance        = CSABasic("HOSCompliance")
	BasicDriverFitness        = CSABasic("DriverFitness")
	BasicControlledSubstances = CSABasic("ControlledSubstances")
	BasicVehicleMaintenance   = CSABasic("VehicleMaintenance")
	BasicHazmatCompliance     = CSABasic("HazmatCompliance")
	BasicCrashIndicator       = CSABasic("CrashIndicator")
)

func (b CSABasic) String() string { return string(b) }

func (b CSABasic) IsValid() bool {
	switch b {
	case BasicUnsafeDriving, BasicHOSCompliance, BasicDriverFitness,
		BasicControlledSubstances, BasicVehicleMaintenance, BasicHazmatCompliance,
		BasicCrashIndicator:
		return true
	default:
		return false
	}
}

// AllCSABasics is the display order the FMCSA itself uses, so a scorecard put
// beside a Safety Measurement System report reads down in the same order.
func AllCSABasics() []CSABasic {
	return []CSABasic{
		BasicUnsafeDriving,
		BasicHOSCompliance,
		BasicDriverFitness,
		BasicControlledSubstances,
		BasicVehicleMaintenance,
		BasicHazmatCompliance,
		BasicCrashIndicator,
	}
}

// SuggestedBasic is where an event lands when nobody has keyed in the
// violations behind it. It is a fallback, not a claim: an inspection can cite
// several BASICs at once and only the violations themselves can say which.
// Without it a carrier who records events but not violation codes would read a
// blank scorecard, which is worse than an approximate one.
func SuggestedBasic(
	kind workersafety.SafetyEventKind,
	result workersafety.InspectionResult,
) CSABasic {
	switch kind {
	case workersafety.SafetyEventAccident:
		return BasicCrashIndicator
	case workersafety.SafetyEventCitation:
		return BasicUnsafeDriving
	case workersafety.SafetyEventInspection:
		if result == workersafety.InspectionResultOutOfService ||
			result == workersafety.InspectionResultFail {
			return BasicVehicleMaintenance
		}
		return ""
	case workersafety.SafetyEventIncident, workersafety.SafetyEventNearMiss:
		return ""
	default:
		return ""
	}
}

// CSA time weights. The FMCSA weights a violation by how recently it happened,
// because a fleet that cleaned up its act a year ago is not the fleet it was.
const (
	csaRecentMonths    = 6
	csaMidMonths       = 12
	csaWeightRecent    = int32(3)
	csaWeightMid       = int32(2)
	csaWeightOld       = int32(1)
	csaOutOfServiceAdd = int32(2)
)

// CSATimeWeight is the multiplier a violation carries at now: three inside six
// months, two inside a year, one out to the two-year look-back, and nothing
// beyond it.
func CSATimeWeight(occurredAt, now int64) int32 {
	if occurredAt > now {
		return csaWeightRecent
	}
	switch {
	case occurredAt >= timeutils.AddMonthsUTC(now, -csaRecentMonths):
		return csaWeightRecent
	case occurredAt >= timeutils.AddMonthsUTC(now, -csaMidMonths):
		return csaWeightMid
	case occurredAt >= timeutils.AddMonthsUTC(now, -workersafety.SafetyPointsRetentionMonths):
		return csaWeightOld
	default:
		return 0
	}
}

var (
	_ validationframework.TenantedEntity = (*WorkerSafetyViolation)(nil)
)

// WorkerSafetyViolation is one violation cited on a safety event. A roadside
// inspection routinely produces several, in different BASICs, so they are rows
// rather than a column on the event.
type WorkerSafetyViolation struct {
	keel.Table `keel:"worker_safety_violations"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:wsvi_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	SafetyEventID  pulid.ID `json:"safetyEventId"  keel:"safety_event_id"`
	// WorkerID is denormalised from the event so a per-driver violation list
	// does not need the join. The event owns the date; nothing here does.
	WorkerID pulid.ID `json:"workerId"       keel:"worker_id"`

	Basic       CSABasic `json:"basic"          keel:"basic"`
	Code        string   `json:"code"           keel:"code,nullzero"`
	Description string   `json:"description"    keel:"description"`
	// SeverityWeight is the FMCSA weight, 1 to 10. Out of service adds two more
	// when the BASIC is scored, which is why the flag stays beside it rather
	// than being folded in — folding it in would make the weight unreadable
	// against the published tables.
	SeverityWeight int16 `json:"severityWeight" keel:"severity_weight"`
	OutOfService   bool  `json:"outOfService"   keel:"out_of_service"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	SafetyEvent *workersafety.WorkerSafetyEvent `json:"safetyEvent,omitempty" keel:"belongs:safety_event_id"`
}

func (v *WorkerSafetyViolation) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(v,
		validation.Field(&v.SafetyEventID, validation.Required.Error("Safety event is required")),
		validation.Field(&v.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&v.Basic,
			validation.Required.Error("BASIC is required"),
			domainvalidation.ValidEnum[CSABasic]("BASIC is not one of the seven"),
		),
		validation.Field(&v.Description,
			validation.Required.Error("Describe the violation"),
			validation.Length(1, 255).Error("Description cannot exceed 255 characters"),
		),
		validation.Field(&v.Code,
			validation.Length(0, 20).Error("Code cannot exceed 20 characters"),
		),
		validation.Field(&v.SeverityWeight,
			validation.Min(int16(1)).Error("Severity weight is 1 to 10"),
			validation.Max(int16(10)).Error("Severity weight is 1 to 10"),
		),
	))
}

// WeightedScore is what the violation contributes to its BASIC at now: the
// severity weight, plus two for an out-of-service order, multiplied by how
// recently it happened.
func (v *WorkerSafetyViolation) WeightedScore(occurredAt, now int64) int32 {
	weight := int32(v.SeverityWeight)
	if v.OutOfService {
		weight += csaOutOfServiceAdd
	}
	return weight * CSATimeWeight(occurredAt, now)
}

func (v *WorkerSafetyViolation) GetID() pulid.ID { return v.ID }

func (v *WorkerSafetyViolation) GetCreatedAt() int64 { return v.CreatedAt }

func (v *WorkerSafetyViolation) GetOrganizationID() pulid.ID { return v.OrganizationID }

func (v *WorkerSafetyViolation) GetBusinessUnitID() pulid.ID { return v.BusinessUnitID }

func (v *WorkerSafetyViolation) GetTableName() string { return "worker_safety_violations" }

func (v *WorkerSafetyViolation) GetResourceType() string { return "worker_safety_violation" }

func (v *WorkerSafetyViolation) GetResourceID() string { return v.ID.String() }

func (v *WorkerSafetyViolation) BeforeCreate(_ context.Context) error {
	if v.SeverityWeight <= 0 {
		v.SeverityWeight = 1
	}
	if v.SeverityWeight == 0 {
		v.SeverityWeight = 1
	}

	return nil
}

func (WorkerSafetyViolation) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_worker_safety_violations_event",
			Message: "Safety event does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_safety_violations_worker",
			Message: "Worker does not exist or belongs to a different organization",
		},
	}
}
