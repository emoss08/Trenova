package jurisdictionrule

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*EscortThreshold)(nil)

// EscortThreshold is a child row rather than a fixed set of columns because the
// escort ladder is genuinely per-jurisdiction: a first escort starts at 12'0"
// in Nebraska, 12'1" in Florida, 14'1" in Texas on divided highways, and not
// until 16'7" in Montana. Columns would force every state into one shape.
type EscortThreshold struct {
	bun.BaseModel `bun:"table:jurisdiction_escort_thresholds,alias:jet" json:"-"`

	ID                 pulid.ID    `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	JurisdictionRuleID pulid.ID    `json:"jurisdictionRuleId" bun:"jurisdiction_rule_id,type:VARCHAR(100),notnull"`
	Role               EscortRole  `json:"role"               bun:"role,type:jurisdiction_escort_role_enum,notnull"`
	Trigger            TriggerKind `json:"trigger"            bun:"trigger,type:jurisdiction_trigger_kind_enum,notnull"`
	ThresholdValue     float64     `json:"thresholdValue"     bun:"threshold_value,type:NUMERIC(12,2),notnull"`
	Note               string      `json:"note"               bun:"note,type:TEXT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	JurisdictionRule *JurisdictionRule `json:"-" bun:"rel:belongs-to,join:jurisdiction_rule_id=id"`
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

func (e *EscortThreshold) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("jet_")
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
}
