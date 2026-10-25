package jurisdictionrule

import (
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// EscortThreshold is a child row rather than a fixed set of columns because the
// escort ladder is genuinely per-jurisdiction: a first escort starts at 12'0"
// in Nebraska, 12'1" in Florida, 14'1" in Texas on divided highways, and not
// until 16'7" in Montana. Columns would force every state into one shape.
type EscortThreshold struct {
	keel.Table `keel:"jurisdiction_escort_thresholds"`

	ID                 pulid.ID    `json:"id"                 keel:"id,pk,autoid:jet_"`
	JurisdictionRuleID pulid.ID    `json:"jurisdictionRuleId" keel:"jurisdiction_rule_id"`
	Role               EscortRole  `json:"role"               keel:"role"`
	Trigger            TriggerKind `json:"trigger"            keel:"trigger"`
	ThresholdValue     float64     `json:"thresholdValue"     keel:"threshold_value"`
	Note               string      `json:"note"               keel:"note,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	JurisdictionRule *JurisdictionRule `json:"-" keel:"belongs:jurisdiction_rule_id"`
}

func (e *EscortThreshold) GetID() pulid.ID { return e.ID }

func (e *EscortThreshold) GetTableName() string { return "jurisdiction_escort_thresholds" }

// Triggered reports whether a measured dimension reaches this threshold.
// Thresholds are inclusive: a state that requires an escort "at 12 feet" means
// a 12'0" load already needs one.
func (e *EscortThreshold) Triggered(measured float64) bool {
	return measured >= e.ThresholdValue
}

func (e *EscortThreshold) Describe() string {
	return fmt.Sprintf(
		"%s required at %.2f %s or more of %s",
		e.Role.Label(),
		e.ThresholdValue,
		e.Trigger.Unit(),
		e.Trigger.String(),
	)
}

func (e *EscortThreshold) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.Role,
			validation.Required.Error("Escort role is required"),
			domainvalidation.ValidEnum[EscortRole]("Escort role is invalid"),
		),
		validation.Field(&e.Trigger,
			validation.Required.Error("Trigger is required"),
			domainvalidation.ValidEnum[TriggerKind]("Trigger is invalid"),
		),
	))

	if e.ThresholdValue <= 0 {
		multiErr.Add("thresholdValue", errortypes.ErrInvalid,
			"Threshold must be greater than zero")
	}
}
