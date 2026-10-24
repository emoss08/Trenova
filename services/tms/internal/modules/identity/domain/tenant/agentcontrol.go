package tenant

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/editchange"
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
	keel.Table    `keel:"agent_controls"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:agc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`

	ShadowMode bool `json:"shadowMode" bun:"shadow_mode,type:BOOLEAN,notnull" keel:"shadow_mode"`

	// EarnedAutonomy lets a streak of clean approvals move a tool up one tier
	// on an agent, and PromotionThreshold is how long that streak must be.
	// Off by default: an agent widening its own reach is something an
	// organization opts into.
	EarnedAutonomy     bool `json:"earnedAutonomy"     bun:"earned_autonomy,type:BOOLEAN,notnull" keel:"earned_autonomy"`
	PromotionThreshold int  `json:"promotionThreshold" bun:"promotion_threshold,type:INTEGER,notnull" keel:"promotion_threshold"`

	// BriefingEnabled turns the morning briefing on, and BriefingHourLocal
	// is the hour it is written in the organization's own timezone. The
	// hour is local rather than UTC because a briefing is read at the
	// start of a working day, and a company with offices in two timezones
	// would otherwise get one of them yesterday's page.
	BriefingEnabled   bool `json:"briefingEnabled"   bun:"briefing_enabled,type:BOOLEAN,notnull" keel:"briefing_enabled"`
	BriefingHourLocal int  `json:"briefingHourLocal" bun:"briefing_hour_local,type:INTEGER,notnull,default:6" keel:"briefing_hour_local"`

	AITrainingConsent            bool      `json:"aiTrainingConsent"            bun:"ai_training_consent,type:BOOLEAN,notnull,default:false" keel:"ai_training_consent"`
	AITrainingConsentChangedAt   *int64    `json:"aiTrainingConsentChangedAt"   bun:"ai_training_consent_changed_at,type:BIGINT,nullzero" keel:"ai_training_consent_changed_at"`
	AITrainingConsentChangedByID *pulid.ID `json:"aiTrainingConsentChangedById" bun:"ai_training_consent_changed_by_id,type:VARCHAR(100),nullzero" keel:"ai_training_consent_changed_by_id"`

	// PersonMonthlyMessages is how many questions one person may ask the
	// agents in a calendar month; nought leaves it unlimited.
	PersonMonthlyMessages int `json:"personMonthlyMessages" bun:"person_monthly_messages,type:INTEGER,notnull,default:0" keel:"person_monthly_messages"`

	LearningOff bool `json:"learningOff" bun:"learning_off,type:BOOLEAN,notnull,default:false" keel:"learning_off"`

	AgentMoneyCallCeilingMinor int64 `json:"agentMoneyCallCeilingMinor" bun:"agent_money_call_ceiling_minor,type:BIGINT,notnull,default:100000" keel:"agent_money_call_ceiling_minor"`
	AgentMoneyRunCalls         int   `json:"agentMoneyRunCalls"         bun:"agent_money_run_calls,type:INTEGER,notnull,default:10" keel:"agent_money_run_calls"`
	AgentMoneyRunAmountMinor   int64 `json:"agentMoneyRunAmountMinor"   bun:"agent_money_run_amount_minor,type:BIGINT,notnull,default:500000" keel:"agent_money_run_amount_minor"`
	AgentMoneyDayCalls         int   `json:"agentMoneyDayCalls"         bun:"agent_money_day_calls,type:INTEGER,notnull,default:50" keel:"agent_money_day_calls"`
	AgentMoneyDayAmountMinor   int64 `json:"agentMoneyDayAmountMinor"   bun:"agent_money_day_amount_minor,type:BIGINT,notnull,default:2500000" keel:"agent_money_day_amount_minor"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

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
	maxPersonMonthlyMessages = 1_000_000

	DefaultAgentMoneyCallCeilingMinor = 100_000
	DefaultAgentMoneyRunCalls         = 10
	DefaultAgentMoneyRunAmountMinor   = 500_000
	DefaultAgentMoneyDayCalls         = 50
	DefaultAgentMoneyDayAmountMinor   = 2_500_000
	maxAgentMoneyAmountMinor          = 100_000_000_000
	maxAgentMoneyCalls                = 10_000
)

func (ac *AgentControl) ApplyAgentMoneyDefaults() {
	ac.AgentMoneyCallCeilingMinor = DefaultAgentMoneyCallCeilingMinor
	ac.AgentMoneyRunCalls = DefaultAgentMoneyRunCalls
	ac.AgentMoneyRunAmountMinor = DefaultAgentMoneyRunAmountMinor
	ac.AgentMoneyDayCalls = DefaultAgentMoneyDayCalls
	ac.AgentMoneyDayAmountMinor = DefaultAgentMoneyDayAmountMinor
}

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
	if ac.PersonMonthlyMessages < 0 || ac.PersonMonthlyMessages > maxPersonMonthlyMessages {
		multiErr.Add(
			"personMonthlyMessages",
			errortypes.ErrInvalid,
			"The monthly allowance must be between 0 (unlimited) and 1,000,000 questions",
		)
	}
	ac.validateAgentMoney(multiErr)
}

func (ac *AgentControl) validateAgentMoney(multiErr *errortypes.MultiError) {
	amounts := []struct {
		field string
		value int64
	}{
		{"agentMoneyCallCeiling", ac.AgentMoneyCallCeilingMinor},
		{"agentMoneyRunAmount", ac.AgentMoneyRunAmountMinor},
		{"agentMoneyDayAmount", ac.AgentMoneyDayAmountMinor},
	}
	for _, amount := range amounts {
		if amount.value < 0 || amount.value > maxAgentMoneyAmountMinor {
			multiErr.Add(
				amount.field,
				errortypes.ErrInvalid,
				"The amount must be between 0 and 1,000,000,000.00",
			)
		}
	}
	calls := []struct {
		field string
		value int
	}{
		{"agentMoneyRunCalls", ac.AgentMoneyRunCalls},
		{"agentMoneyDayCalls", ac.AgentMoneyDayCalls},
	}
	for _, count := range calls {
		if count.value < 0 || count.value > maxAgentMoneyCalls {
			multiErr.Add(
				count.field,
				errortypes.ErrInvalid,
				"The number of changes must be between 0 and 10,000",
			)
		}
	}
	if ac.AgentMoneyRunAmountMinor > ac.AgentMoneyDayAmountMinor {
		multiErr.Add(
			"agentMoneyRunAmount",
			errortypes.ErrInvalid,
			"A run's total cannot be more than the day's",
		)
	}
	if ac.AgentMoneyRunCalls > ac.AgentMoneyDayCalls {
		multiErr.Add(
			"agentMoneyRunCalls",
			errortypes.ErrInvalid,
			"A run's number of changes cannot be more than the day's",
		)
	}
	if ac.AgentMoneyCallCeilingMinor > ac.AgentMoneyRunAmountMinor {
		multiErr.Add(
			"agentMoneyCallCeiling",
			errortypes.ErrInvalid,
			"One change cannot be more than a run's total",
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

// AgentControlChangeRules name, in the organization-wide editor's words, the
// settings a person edits there.
var AgentControlChangeRules = []editchange.Rule[AgentControl]{
	{
		Field: "shadowMode", Label: "Pause every agent",
		Same: func(a, b *AgentControl) bool { return a.ShadowMode == b.ShadowMode },
	},
	{
		Field: "earnedAutonomy", Label: "Earned autonomy",
		Same: func(a, b *AgentControl) bool {
			return a.EarnedAutonomy == b.EarnedAutonomy &&
				a.PromotionThreshold == b.PromotionThreshold
		},
	},
	{
		Field: "learningOff", Label: "Learn from their work",
		Same: func(a, b *AgentControl) bool { return a.LearningOff == b.LearningOff },
	},
	{
		Field: "personMonthlyMessages", Label: "Monthly allowance",
		Same: func(a, b *AgentControl) bool {
			return a.PersonMonthlyMessages == b.PersonMonthlyMessages
		},
	},
	{
		Field: "agentMoneyLimits", Label: "Money limits for agents",
		Same: func(a, b *AgentControl) bool {
			return a.AgentMoneyCallCeilingMinor == b.AgentMoneyCallCeilingMinor &&
				a.AgentMoneyRunCalls == b.AgentMoneyRunCalls &&
				a.AgentMoneyRunAmountMinor == b.AgentMoneyRunAmountMinor &&
				a.AgentMoneyDayCalls == b.AgentMoneyDayCalls &&
				a.AgentMoneyDayAmountMinor == b.AgentMoneyDayAmountMinor
		},
	},
	{
		Field: "aiTrainingConsent", Label: "Share corrections",
		Same: func(a, b *AgentControl) bool { return a.AITrainingConsent == b.AITrainingConsent },
	},
}
