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
func SuggestedBasic(kind SafetyEventKind, result InspectionResult) CSABasic {
	switch kind {
	case SafetyEventAccident:
		return BasicCrashIndicator
	case SafetyEventCitation:
		return BasicUnsafeDriving
	case SafetyEventInspection:
		if result == InspectionResultOutOfService || result == InspectionResultFail {
			return BasicVehicleMaintenance
		}
		return ""
	case SafetyEventIncident, SafetyEventNearMiss:
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
	case occurredAt >= timeutils.AddMonthsUTC(now, -SafetyPointsRetentionMonths):
		return csaWeightOld
	default:
		return 0
	}
}

var (
	_ bun.BeforeAppendModelHook          = (*WorkerSafetyViolation)(nil)
	_ validationframework.TenantedEntity = (*WorkerSafetyViolation)(nil)
)

// WorkerSafetyViolation is one violation cited on a safety event. A roadside
// inspection routinely produces several, in different BASICs, so they are rows
// rather than a column on the event.
type WorkerSafetyViolation struct {
	bun.BaseModel `bun:"table:worker_safety_violations,alias:wsvi" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	SafetyEventID  pulid.ID `json:"safetyEventId"  bun:"safety_event_id,type:VARCHAR(100),notnull"`
	// WorkerID is denormalised from the event so a per-driver violation list
	// does not need the join. The event owns the date; nothing here does.
	WorkerID pulid.ID `json:"workerId" bun:"worker_id,type:VARCHAR(100),notnull"`

	Basic       CSABasic `json:"basic"       bun:"basic,type:csa_basic_enum,notnull"`
	Code        string   `json:"code"        bun:"code,type:VARCHAR(20),nullzero"`
	Description string   `json:"description" bun:"description,type:VARCHAR(255),notnull"`
	// SeverityWeight is the FMCSA weight, 1 to 10. Out of service adds two more
	// when the BASIC is scored, which is why the flag stays beside it rather
	// than being folded in — folding it in would make the weight unreadable
	// against the published tables.
	SeverityWeight int16 `json:"severityWeight" bun:"severity_weight,type:SMALLINT,notnull,default:1"`
	OutOfService   bool  `json:"outOfService"   bun:"out_of_service,type:BOOLEAN,notnull"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	SafetyEvent *WorkerSafetyEvent `json:"safetyEvent,omitempty" bun:"rel:belongs-to,join:safety_event_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (v *WorkerSafetyViolation) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if v.ID.IsNil() {
			v.ID = pulid.MustNew("wsvi_")
		}
		if v.SeverityWeight <= 0 {
			v.SeverityWeight = 1
		}
		v.CreatedAt = now
		v.UpdatedAt = now
	case *bun.UpdateQuery:
		v.UpdatedAt = now
	}

	return nil
}
