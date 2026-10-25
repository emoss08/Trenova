package tenant

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/editchange"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	_ validationframework.TenantedEntity = (*AgentControl)(nil)
)

type AgentControl struct {
	keel.Table `keel:"agent_controls"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:agc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	ShadowMode bool `json:"shadowMode" keel:"shadow_mode"`

	// EarnedAutonomy lets a streak of clean approvals move a tool up one tier
	// on an agent, and PromotionThreshold is how long that streak must be.
	// Off by default: an agent widening its own reach is something an
	// organization opts into.
	EarnedAutonomy     bool `json:"earnedAutonomy"     keel:"earned_autonomy"`
	PromotionThreshold int  `json:"promotionThreshold" keel:"promotion_threshold"`

	// BriefingEnabled turns the morning briefing on, and BriefingHourLocal
	// is the hour it is written in the organization's own timezone. The
	// hour is local rather than UTC because a briefing is read at the
	// start of a working day, and a company with offices in two timezones
	// would otherwise get one of them yesterday's page.
	BriefingEnabled   bool `json:"briefingEnabled"   keel:"briefing_enabled"`
	BriefingHourLocal int  `json:"briefingHourLocal" keel:"briefing_hour_local"`

	AITrainingConsent            bool      `json:"aiTrainingConsent"            keel:"ai_training_consent"`
	AITrainingConsentChangedAt   *int64    `json:"aiTrainingConsentChangedAt"   keel:"ai_training_consent_changed_at"`
	AITrainingConsentChangedByID *pulid.ID `json:"aiTrainingConsentChangedById" keel:"ai_training_consent_changed_by_id"`

	// PersonMonthlyMessages is how many questions one person may ask the
	// agents in a calendar month; nought leaves it unlimited.
	PersonMonthlyMessages int `json:"personMonthlyMessages" keel:"person_monthly_messages"`

	LearningOff bool `json:"learningOff" keel:"learning_off"`

	AgentMoneyCallCeilingMinor int64 `json:"agentMoneyCallCeilingMinor" keel:"agent_money_call_ceiling_minor"`
	AgentMoneyRunCalls         int   `json:"agentMoneyRunCalls"         keel:"agent_money_run_calls"`
	AgentMoneyRunAmountMinor   int64 `json:"agentMoneyRunAmountMinor"   keel:"agent_money_run_amount_minor"`
	AgentMoneyDayCalls         int   `json:"agentMoneyDayCalls"         keel:"agent_money_day_calls"`
	AgentMoneyDayAmountMinor   int64 `json:"agentMoneyDayAmountMinor"   keel:"agent_money_day_amount_minor"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty"`
	Organization *Organization `json:"organization,omitempty"`
}

func (ac *AgentControl) BeforeCreate(_ context.Context) error {
	if ac.BriefingHourLocal == 0 {
		ac.BriefingHourLocal = 6
	}
	if ac.AgentMoneyCallCeilingMinor == 0 {
		ac.AgentMoneyCallCeilingMinor = 100000
	}
	if ac.AgentMoneyRunCalls == 0 {
		ac.AgentMoneyRunCalls = 10
	}
	if ac.AgentMoneyRunAmountMinor == 0 {
		ac.AgentMoneyRunAmountMinor = 500000
	}
	if ac.AgentMoneyDayCalls == 0 {
		ac.AgentMoneyDayCalls = 50
	}
	if ac.AgentMoneyDayAmountMinor == 0 {
		ac.AgentMoneyDayAmountMinor = 2500000
	}

	return nil
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
