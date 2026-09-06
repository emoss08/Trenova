package worker

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	ErrInvalidPTOPolicyStatus  = errors.New("invalid PTO policy status")
	ErrInvalidPTOYearBasis     = errors.New("invalid PTO year basis")
	ErrInvalidPTOAccrualMethod = errors.New("invalid PTO accrual method")
)

type PTOPolicyStatus string

const (
	PTOPolicyStatusActive   = PTOPolicyStatus("Active")
	PTOPolicyStatusInactive = PTOPolicyStatus("Inactive")
	PTOPolicyStatusDraft    = PTOPolicyStatus("Draft")
)

func (s PTOPolicyStatus) String() string { return string(s) }

func (s PTOPolicyStatus) IsValid() bool {
	switch s {
	case PTOPolicyStatusActive, PTOPolicyStatusInactive, PTOPolicyStatusDraft:
		return true
	default:
		return false
	}
}

func PTOPolicyStatusFromString(s string) (PTOPolicyStatus, error) {
	status := PTOPolicyStatus(s)
	if !status.IsValid() {
		return "", ErrInvalidPTOPolicyStatus
	}
	return status, nil
}

type PTOYearBasis string

const (
	PTOYearBasisCalendarYear    = PTOYearBasis("CalendarYear")
	PTOYearBasisHireAnniversary = PTOYearBasis("HireAnniversary")
)

func (b PTOYearBasis) String() string { return string(b) }

func (b PTOYearBasis) IsValid() bool {
	switch b {
	case PTOYearBasisCalendarYear, PTOYearBasisHireAnniversary:
		return true
	default:
		return false
	}
}

type PTOAccrualMethod string

const (
	PTOAccrualMethodNone             = PTOAccrualMethod("None")
	PTOAccrualMethodFixedAnnualGrant = PTOAccrualMethod("FixedAnnualGrant")
	PTOAccrualMethodMonthly          = PTOAccrualMethod("Monthly")
	PTOAccrualMethodPerPayPeriod     = PTOAccrualMethod("PerPayPeriod")
)

func (m PTOAccrualMethod) String() string { return string(m) }

func (m PTOAccrualMethod) IsValid() bool {
	switch m {
	case PTOAccrualMethodNone, PTOAccrualMethodFixedAnnualGrant, PTOAccrualMethodMonthly,
		PTOAccrualMethodPerPayPeriod:
		return true
	default:
		return false
	}
}

var ErrInvalidPTOTerminationAction = errors.New("invalid PTO termination action")

// PTOTerminationAction decides what happens to a positive balance when the
// worker's employment ends.
type PTOTerminationAction string

const (
	PTOTerminationForfeit = PTOTerminationAction("Forfeit")
	PTOTerminationPayOut  = PTOTerminationAction("PayOut")
)

func (a PTOTerminationAction) String() string { return string(a) }

func (a PTOTerminationAction) IsValid() bool {
	switch a {
	case PTOTerminationForfeit, PTOTerminationPayOut:
		return true
	default:
		return false
	}
}

// PTOAccrualTier raises the accrual (and optionally the cap) once a worker
// has served MinMonths. Tiers are kept sorted by MinMonths ascending; the
// rule's base amount applies below the first tier.
type PTOAccrualTier struct {
	MinMonths         int32               `json:"minMonths"`
	AccrualAmountDays decimal.Decimal     `json:"accrualAmountDays"`
	MaxBalanceDays    decimal.NullDecimal `json:"maxBalanceDays"`
}

var (
	_ bun.BeforeAppendModelHook          = (*PTOPolicy)(nil)
	_ domaintypes.PostgresSearchable     = (*PTOPolicy)(nil)
	_ pagination.CursorEntity            = (*PTOPolicy)(nil)
	_ validationframework.TenantedEntity = (*PTOPolicy)(nil)
	_ bun.BeforeAppendModelHook          = (*PTOPolicyRule)(nil)
)

type PTOPolicy struct {
	bun.BaseModel             `bun:"table:pto_policies,alias:ptop" json:"-"`
	pagination.CursorValueSet `bun:",embed"                        json:"-"`

	ID                pulid.ID        `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID        `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID        `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Name              string          `json:"name"              bun:"name,type:VARCHAR(100),notnull"`
	Code              string          `json:"code"              bun:"code,type:VARCHAR(50),notnull"`
	Description       string          `json:"description"       bun:"description,type:TEXT,nullzero"`
	Status            PTOPolicyStatus `json:"status"            bun:"status,type:pto_policy_status_enum,notnull,default:'Draft'"`
	IsDefault         bool            `json:"isDefault"         bun:"is_default,type:BOOLEAN,notnull"`
	YearBasis         PTOYearBasis    `json:"yearBasis"         bun:"year_basis,type:pto_year_basis_enum,notnull,default:'CalendarYear'"`
	CountWeekends     bool            `json:"countWeekends"     bun:"count_weekends,type:BOOLEAN,notnull,default:true"`
	WaitingPeriodDays int32           `json:"waitingPeriodDays" bun:"waiting_period_days,type:INTEGER,notnull"`
	RequiresApproval  bool            `json:"requiresApproval"  bun:"requires_approval,type:BOOLEAN,notnull,default:true"`
	EnforceBalance    bool            `json:"enforceBalance"    bun:"enforce_balance,type:BOOLEAN,notnull,default:true"`
	AllowNegative     bool            `json:"allowNegative"     bun:"allow_negative,type:BOOLEAN,notnull"`
	NegativeFloorDays decimal.Decimal `json:"negativeFloorDays" bun:"negative_floor_days,type:NUMERIC(6,2),notnull,default:0"`
	Version           int64           `json:"version"           bun:"version,type:BIGINT"`
	CreatedAt         int64           `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64           `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector      string          `json:"-"                 bun:"search_vector,type:TSVECTOR,scanonly"`

	Rules []*PTOPolicyRule `json:"rules,omitempty" bun:"rel:has-many,join:id=pto_policy_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (p *PTOPolicy) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&p.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 50).Error("Code must be between 1 and 50 characters"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[PTOPolicyStatus](
				"status must be one of: Active, Inactive, Draft",
			),
		),
		validation.Field(&p.YearBasis,
			validation.Required.Error("Year basis is required"),
			domainvalidation.ValidEnum[PTOYearBasis](
				"yearBasis must be one of: CalendarYear, HireAnniversary",
			),
		),
		validation.Field(&p.WaitingPeriodDays,
			validation.Min(int32(0)).Error("Waiting period cannot be negative"),
		),
	))

	if p.NegativeFloorDays.IsPositive() {
		multiErr.Add(
			"negativeFloorDays",
			errortypes.ErrInvalid,
			"Negative floor must be zero or below",
		)
	}
	if p.AllowNegative && !p.NegativeFloorDays.IsNegative() {
		multiErr.Add(
			"negativeFloorDays",
			errortypes.ErrInvalid,
			"Set how far below zero a balance may go when negative balances are allowed",
		)
	}
	if !p.AllowNegative && !p.NegativeFloorDays.IsZero() {
		multiErr.Add(
			"negativeFloorDays",
			errortypes.ErrInvalid,
			"Negative floor only applies when negative balances are allowed",
		)
	}

	seen := make(map[PTOType]struct{}, len(p.Rules))
	for i, rule := range p.Rules {
		if rule == nil {
			continue
		}
		prefix := "rules[" + strconv.Itoa(i) + "]"
		if _, dup := seen[rule.PTOType]; dup {
			multiErr.Add(
				prefix+".ptoType",
				errortypes.ErrDuplicate,
				"Each PTO type may only have one rule",
			)
		}
		seen[rule.PTOType] = struct{}{}
		rule.Validate(multiErr.WithPrefix(prefix))
	}
}

func (p *PTOPolicy) RuleFor(ptoType PTOType) *PTOPolicyRule {
	for _, rule := range p.Rules {
		if rule != nil && rule.PTOType == ptoType {
			return rule
		}
	}
	return nil
}

func (p *PTOPolicy) IsActive() bool {
	return p.Status == PTOPolicyStatusActive
}

func (p *PTOPolicy) GetID() pulid.ID { return p.ID }

func (p *PTOPolicy) GetCreatedAt() int64 { return p.CreatedAt }

func (p *PTOPolicy) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *PTOPolicy) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *PTOPolicy) GetTableName() string { return "pto_policies" }

func (p *PTOPolicy) GetResourceType() string { return "pto_policy" }

func (p *PTOPolicy) GetResourceID() string { return p.ID.String() }

func (p *PTOPolicy) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ptop",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (p *PTOPolicy) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("ptop_")
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}

type PTOPolicyRule struct {
	bun.BaseModel `bun:"table:pto_policy_rules,alias:ptpr" json:"-"`

	ID                  pulid.ID             `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID             `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID             `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	PTOPolicyID         pulid.ID             `json:"ptoPolicyId"         bun:"pto_policy_id,type:VARCHAR(100),notnull"`
	PTOType             PTOType              `json:"ptoType"             bun:"pto_type,type:worker_pto_type_enum,notnull"`
	AccrualMethod       PTOAccrualMethod     `json:"accrualMethod"       bun:"accrual_method,type:pto_accrual_method_enum,notnull,default:'None'"`
	AccrualAmountDays   decimal.Decimal      `json:"accrualAmountDays"   bun:"accrual_amount_days,type:NUMERIC(6,2),notnull,default:0"`
	MaxBalanceDays      decimal.NullDecimal  `json:"maxBalanceDays"      bun:"max_balance_days,type:NUMERIC(6,2),nullzero"`
	CarryoverCapDays    decimal.NullDecimal  `json:"carryoverCapDays"    bun:"carryover_cap_days,type:NUMERIC(6,2),nullzero"`
	CarryoverExpiryDays int32                `json:"carryoverExpiryDays" bun:"carryover_expiry_days,type:INTEGER,notnull"`
	Tiers               []PTOAccrualTier     `json:"tiers"               bun:"tiers,type:JSONB,notnull,default:'[]'"`
	OnTermination       PTOTerminationAction `json:"onTermination"       bun:"on_termination,type:pto_termination_action_enum,notnull,default:'Forfeit'"`
	SortOrder           int32                `json:"sortOrder"           bun:"sort_order,type:INTEGER,notnull"`
	CreatedAt           int64                `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64                `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (r *PTOPolicyRule) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.PTOType,
			validation.Required.Error("PTO type is required"),
			domainvalidation.ValidEnum[PTOType](
				"ptoType must be one of: Personal, Vacation, Sick, Holiday, Bereavement, Maternity, Paternity",
			),
		),
		validation.Field(&r.AccrualMethod,
			validation.Required.Error("Accrual method is required"),
			domainvalidation.ValidEnum[PTOAccrualMethod](
				"accrualMethod must be one of: None, FixedAnnualGrant, Monthly, PerPayPeriod",
			),
		),
		validation.Field(&r.CarryoverExpiryDays,
			validation.Min(int32(0)).Error("Carryover expiry cannot be negative"),
		),
	))

	if r.AccrualAmountDays.IsNegative() {
		multiErr.Add(
			"accrualAmountDays",
			errortypes.ErrInvalid,
			"Accrual amount cannot be negative",
		)
	}
	if r.AccrualMethod != PTOAccrualMethodNone && !r.AccrualAmountDays.IsPositive() {
		multiErr.Add(
			"accrualAmountDays",
			errortypes.ErrRequired,
			"Accrual amount is required when a rule accrues",
		)
	}
	if r.AccrualMethod == PTOAccrualMethodNone && !r.AccrualAmountDays.IsZero() {
		multiErr.Add(
			"accrualAmountDays",
			errortypes.ErrInvalid,
			"Accrual amount must be zero when the rule does not accrue",
		)
	}
	if r.MaxBalanceDays.Valid && !r.MaxBalanceDays.Decimal.IsPositive() {
		multiErr.Add("maxBalanceDays", errortypes.ErrInvalid, "Maximum balance must be above zero")
	}
	if r.CarryoverCapDays.Valid && r.CarryoverCapDays.Decimal.IsNegative() {
		multiErr.Add("carryoverCapDays", errortypes.ErrInvalid, "Carryover cap cannot be negative")
	}
	if r.CarryoverExpiryDays > 0 && !r.CarryoverCapDays.Valid {
		multiErr.Add(
			"carryoverExpiryDays",
			errortypes.ErrInvalid,
			"Carryover expiry needs a carryover cap so the carried amount is known",
		)
	}
	if r.OnTermination != "" && !r.OnTermination.IsValid() {
		multiErr.Add("onTermination", errortypes.ErrInvalid, "onTermination must be Forfeit or PayOut")
	}
	r.validateTiers(multiErr)
}

// TerminationAction is the effective action, defaulting to Forfeit for rules
// written before the setting existed.
func (r *PTOPolicyRule) TerminationAction() PTOTerminationAction {
	if r.OnTermination == "" {
		return PTOTerminationForfeit
	}
	return r.OnTermination
}

func (r *PTOPolicyRule) validateTiers(multiErr *errortypes.MultiError) {
	if len(r.Tiers) > 0 && r.AccrualMethod == PTOAccrualMethodNone {
		multiErr.Add("tiers", errortypes.ErrInvalid, "Tiers only apply to rules that accrue")
	}
	lastMin := int32(0)
	for i, tier := range r.Tiers {
		prefix := "tiers[" + strconv.Itoa(i) + "]"
		if tier.MinMonths <= 0 {
			multiErr.Add(prefix+".minMonths", errortypes.ErrInvalid, "Tier must start after month 0")
		}
		if i > 0 && tier.MinMonths <= lastMin {
			multiErr.Add(prefix+".minMonths", errortypes.ErrInvalid, "Tiers must be in ascending order of tenure")
		}
		lastMin = tier.MinMonths
		if !tier.AccrualAmountDays.IsPositive() {
			multiErr.Add(prefix+".accrualAmountDays", errortypes.ErrInvalid, "Tier accrual must be above zero")
		}
		if tier.MaxBalanceDays.Valid && !tier.MaxBalanceDays.Decimal.IsPositive() {
			multiErr.Add(prefix+".maxBalanceDays", errortypes.ErrInvalid, "Tier maximum balance must be above zero")
		}
	}
}

func (r *PTOPolicyRule) Accrues() bool {
	return r.AccrualMethod != PTOAccrualMethodNone && r.AccrualAmountDays.IsPositive()
}

// TierFor returns the highest tier the tenure has reached, or nil below the
// first tier.
func (r *PTOPolicyRule) TierFor(tenureMonths int32) *PTOAccrualTier {
	var match *PTOAccrualTier
	for i := range r.Tiers {
		if r.Tiers[i].MinMonths <= tenureMonths {
			match = &r.Tiers[i]
		}
	}
	return match
}

// AmountFor is the accrual amount at a given tenure.
func (r *PTOPolicyRule) AmountFor(tenureMonths int32) decimal.Decimal {
	if tier := r.TierFor(tenureMonths); tier != nil {
		return tier.AccrualAmountDays
	}
	return r.AccrualAmountDays
}

// MaxBalanceFor is the balance cap at a given tenure; a tier without its own
// cap inherits the rule's.
func (r *PTOPolicyRule) MaxBalanceFor(tenureMonths int32) decimal.NullDecimal {
	if tier := r.TierFor(tenureMonths); tier != nil && tier.MaxBalanceDays.Valid {
		return tier.MaxBalanceDays
	}
	return r.MaxBalanceDays
}

// TenureMonths counts whole months of service from hire to the given day.
func TenureMonths(hireDate, asOf int64, loc *time.Location) int32 {
	if hireDate <= 0 || asOf <= hireDate {
		return 0
	}
	if loc == nil {
		loc = time.UTC
	}
	hire := localDate(hireDate, loc)
	day := localDate(asOf, loc)
	months := (day.Year()-hire.Year())*12 + int(day.Month()) - int(hire.Month())
	if day.Day() < hire.Day() {
		months--
	}
	if months < 0 {
		return 0
	}
	return int32(months) //nolint:gosec // months of service fit comfortably
}

func (r *PTOPolicyRule) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("ptpr_")
		}
		if r.Tiers == nil {
			r.Tiers = []PTOAccrualTier{}
		}
		if r.OnTermination == "" {
			r.OnTermination = PTOTerminationForfeit
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		if r.Tiers == nil {
			r.Tiers = []PTOAccrualTier{}
		}
		r.UpdatedAt = now
	}

	return nil
}
