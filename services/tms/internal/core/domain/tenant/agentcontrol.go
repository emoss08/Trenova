package tenant

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*AgentControl)(nil)
	_ validationframework.TenantedEntity = (*AgentControl)(nil)
)

type AgentControl struct {
	bun.BaseModel `json:"-" bun:"table:agent_controls,alias:agc"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`

	ShadowMode bool `json:"shadowMode" bun:"shadow_mode,type:BOOLEAN,notnull"`

	// EarnedAutonomy lets a streak of clean approvals move a tool up one tier
	// on an agent, and PromotionThreshold is how long that streak must be.
	// Off by default: an agent widening its own reach is something an
	// organization opts into.
	EarnedAutonomy     bool `json:"earnedAutonomy"     bun:"earned_autonomy,type:BOOLEAN,notnull"`
	PromotionThreshold int  `json:"promotionThreshold" bun:"promotion_threshold,type:INTEGER,notnull"`

	// BriefingEnabled turns the morning briefing on, and BriefingHourLocal
	// is the hour it is written in the organization's own timezone. The
	// hour is local rather than UTC because a briefing is read at the
	// start of a working day, and a company with offices in two timezones
	// would otherwise get one of them yesterday's page.
	BriefingEnabled   bool `json:"briefingEnabled"   bun:"briefing_enabled,type:BOOLEAN,notnull"`
	BriefingHourLocal int  `json:"briefingHourLocal" bun:"briefing_hour_local,type:INTEGER,notnull,default:6"`

	AITrainingConsent            bool      `json:"aiTrainingConsent"            bun:"ai_training_consent,type:BOOLEAN,notnull,default:false"`
	AITrainingConsentChangedAt   *int64    `json:"aiTrainingConsentChangedAt"   bun:"ai_training_consent_changed_at,type:BIGINT,nullzero"`
	AITrainingConsentChangedByID *pulid.ID `json:"aiTrainingConsentChangedById" bun:"ai_training_consent_changed_by_id,type:VARCHAR(100),nullzero"`

	BillingAgentEnabled    bool `json:"billingAgentEnabled"    bun:"-"`
	DecisionTimeoutSeconds int  `json:"decisionTimeoutSeconds" bun:"-"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

const (
	DefaultPromotionThreshold = 10
	minPromotionThreshold     = 1
	maxPromotionThreshold     = 1000
	// DefaultBriefingHourLocal is six in the morning: early enough that
	// the page is waiting when the first dispatcher opens it, late enough
	// that the night's work is in the numbers.
	DefaultBriefingHourLocal = 6
	maxBriefingHour          = 23
)

func (ac *AgentControl) Validate(multiErr *errortypes.MultiError) {
	if ac.PromotionThreshold < minPromotionThreshold ||
		ac.PromotionThreshold > maxPromotionThreshold {
		multiErr.Add(
			"promotionThreshold",
			errortypes.ErrInvalid,
			"Promotion threshold must be between 1 and 1000 approvals",
		)
	}
	if ac.BriefingHourLocal < 0 || ac.BriefingHourLocal > maxBriefingHour {
		multiErr.Add(
			"briefingHourLocal",
			errortypes.ErrInvalid,
			"The briefing hour must be between 0 and 23",
		)
	}
}

func (ac *AgentControl) SetAITrainingConsent(consent bool, changedByID pulid.ID, at int64) bool {
	if ac.AITrainingConsent == consent {
		return false
	}

	ac.AITrainingConsent = consent
	ac.AITrainingConsentChangedAt = &at
	ac.AITrainingConsentChangedByID = &changedByID

	return true
}

func (ac *AgentControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if ac.ID.IsNil() {
			ac.ID = pulid.MustNew("agc_")
		}
		ac.CreatedAt = now
	case *bun.UpdateQuery:
		ac.UpdatedAt = now
	}

	return nil
}

func (ac *AgentControl) GetID() pulid.ID {
	return ac.ID
}

func (ac *AgentControl) GetTableName() string {
	return "agent_controls"
}

func (ac *AgentControl) GetOrganizationID() pulid.ID {
	return ac.OrganizationID
}

func (ac *AgentControl) GetBusinessUnitID() pulid.ID {
	return ac.BusinessUnitID
}
