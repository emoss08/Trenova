package jurisdictionrule

import (
	"fmt"
	"sort"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// Resolved is a jurisdiction rule merged with a carrier's override, carrying
// enough provenance that any derived requirement can explain which source set
// each number.
type Resolved struct {
	RuleID            pulid.ID          `json:"ruleId"`
	StateID           pulid.ID          `json:"stateId"`
	StateCode         string            `json:"stateCode"`
	StateName         string            `json:"stateName"`
	VerificationState VerificationState `json:"verificationState"`
	VerifiedAt        *int64            `json:"verifiedAt,omitempty"`
	SourceNote        string            `json:"sourceNote,omitempty"`
	SourceURL         string            `json:"sourceUrl,omitempty"`

	MaxWidthFeet    float64 `json:"maxWidthFeet"`
	MaxHeightFeet   float64 `json:"maxHeightFeet"`
	MaxLengthFeet   float64 `json:"maxLengthFeet"`
	MaxWeightPounds int64   `json:"maxWeightPounds"`

	SuperloadWidthFeet    *float64 `json:"superloadWidthFeet,omitempty"`
	SuperloadWeightPounds *int64   `json:"superloadWeightPounds,omitempty"`

	DaylightOnly       bool `json:"daylightOnly"`
	RushHourRestricted bool `json:"rushHourRestricted"`
	WeekendRestricted  bool `json:"weekendRestricted"`
	HolidayRestricted  bool `json:"holidayRestricted"`

	PermitLeadTimeDays int16               `json:"permitLeadTimeDays"`
	PermitValidityDays int16               `json:"permitValidityDays"`
	PermitBaseFee      decimal.NullDecimal `json:"permitBaseFee"`
	PermitPerMileFee   decimal.NullDecimal `json:"permitPerMileFee"`

	EscortThresholds []*EscortThreshold `json:"escortThresholds,omitempty"`

	OverriddenFields []string `json:"overriddenFields,omitempty"`
	OverrideReason   string   `json:"overrideReason,omitempty"`
}

func (r *JurisdictionRule) Resolve(override *Override) *Resolved {
	resolved := &Resolved{
		RuleID:                r.ID,
		StateID:               r.StateID,
		VerificationState:     r.VerificationState,
		VerifiedAt:            r.VerifiedAt,
		SourceNote:            r.SourceNote,
		SourceURL:             r.SourceURL,
		MaxWidthFeet:          r.MaxWidthFeet,
		MaxHeightFeet:         r.MaxHeightFeet,
		MaxLengthFeet:         r.MaxLengthFeet,
		MaxWeightPounds:       r.MaxWeightPounds,
		SuperloadWidthFeet:    r.SuperloadWidthFeet,
		SuperloadWeightPounds: r.SuperloadWeightPounds,
		DaylightOnly:          r.DaylightOnly,
		RushHourRestricted:    r.RushHourRestricted,
		WeekendRestricted:     r.WeekendRestricted,
		HolidayRestricted:     r.HolidayRestricted,
		PermitLeadTimeDays:    r.PermitLeadTimeDays,
		PermitValidityDays:    r.PermitValidityDays,
		PermitBaseFee:         r.PermitBaseFee,
		PermitPerMileFee:      r.PermitPerMileFee,
		EscortThresholds:      r.EscortThresholds,
	}

	if r.State != nil {
		resolved.StateCode = r.State.Abbreviation
		resolved.StateName = r.State.Name
	}

	if override == nil {
		return resolved
	}

	resolved.OverrideReason = override.Reason
	resolved.OverriddenFields = make([]string, 0, 7)

	applyFloat(&resolved.MaxWidthFeet, override.MaxWidthFeet,
		"maxWidthFeet", &resolved.OverriddenFields)
	applyFloat(&resolved.MaxHeightFeet, override.MaxHeightFeet,
		"maxHeightFeet", &resolved.OverriddenFields)
	applyFloat(&resolved.MaxLengthFeet, override.MaxLengthFeet,
		"maxLengthFeet", &resolved.OverriddenFields)

	if override.MaxWeightPounds != nil {
		resolved.MaxWeightPounds = *override.MaxWeightPounds
		resolved.OverriddenFields = append(resolved.OverriddenFields, "maxWeightPounds")
	}
	if override.PermitLeadTimeDays != nil {
		resolved.PermitLeadTimeDays = *override.PermitLeadTimeDays
		resolved.OverriddenFields = append(resolved.OverriddenFields, "permitLeadTimeDays")
	}
	if override.DaylightOnly != nil {
		resolved.DaylightOnly = *override.DaylightOnly
		resolved.OverriddenFields = append(resolved.OverriddenFields, "daylightOnly")
	}
	if override.HolidayRestricted != nil {
		resolved.HolidayRestricted = *override.HolidayRestricted
		resolved.OverriddenFields = append(resolved.OverriddenFields, "holidayRestricted")
	}

	if len(resolved.OverriddenFields) == 0 {
		resolved.OverriddenFields = nil
		resolved.OverrideReason = ""
	}

	return resolved
}

func applyFloat(target *float64, override *float64, field string, applied *[]string) {
	if override == nil {
		return
	}
	*target = *override
	*applied = append(*applied, field)
}

// Measurements is the load presented to a jurisdiction for evaluation.
type Measurements struct {
	WidthFeet     float64
	HeightFeet    float64
	LengthFeet    float64
	WeightPounds  int64
	DistanceMiles float64
}

type Exceedance struct {
	Trigger   TriggerKind `json:"trigger"`
	Limit     float64     `json:"limit"`
	Actual    float64     `json:"actual"`
	OverBy    float64     `json:"overBy"`
	Superload bool        `json:"superload"`
}

func (e Exceedance) Describe(stateCode string) string {
	return fmt.Sprintf(
		"%s is %.2f %s, over the %s limit of %.2f %s by %.2f",
		e.Trigger.String(), e.Actual, e.Trigger.Unit(),
		stateCode, e.Limit, e.Trigger.Unit(), e.OverBy,
	)
}

type EscortRequirement struct {
	Role      EscortRole  `json:"role"`
	Trigger   TriggerKind `json:"trigger"`
	Threshold float64     `json:"threshold"`
	Actual    float64     `json:"actual"`
	Note      string      `json:"note,omitempty"`
}

// Evaluate reports every way a load exceeds this jurisdiction. It returns all
// exceedances rather than the first, because a load can be simultaneously over
// width and over height and the operator needs both numbers to file correctly.
func (r *Resolved) Evaluate(m Measurements) []Exceedance {
	exceedances := make([]Exceedance, 0, 4)

	add := func(trigger TriggerKind, limit, actual float64) {
		if actual > limit {
			exceedances = append(exceedances, Exceedance{
				Trigger: trigger,
				Limit:   limit,
				Actual:  actual,
				OverBy:  round2(actual - limit),
			})
		}
	}

	add(TriggerWidth, r.MaxWidthFeet, m.WidthFeet)
	add(TriggerHeight, r.MaxHeightFeet, m.HeightFeet)
	add(TriggerLength, r.MaxLengthFeet, m.LengthFeet)

	if m.WeightPounds > r.MaxWeightPounds {
		exceedances = append(exceedances, Exceedance{
			Trigger: TriggerWeight,
			Limit:   float64(r.MaxWeightPounds),
			Actual:  float64(m.WeightPounds),
			OverBy:  float64(m.WeightPounds - r.MaxWeightPounds),
		})
	}

	for idx := range exceedances {
		exceedances[idx].Superload = r.isSuperload(exceedances[idx], m)
	}

	return exceedances
}

func (r *Resolved) isSuperload(e Exceedance, m Measurements) bool {
	switch e.Trigger {
	case TriggerWidth:
		return r.SuperloadWidthFeet != nil && m.WidthFeet >= *r.SuperloadWidthFeet
	case TriggerWeight:
		return r.SuperloadWeightPounds != nil && m.WeightPounds >= *r.SuperloadWeightPounds
	case TriggerHeight, TriggerLength, TriggerSuperload:
		return false
	default:
		return false
	}
}

func (r *Resolved) RequiresPermit(m Measurements) bool {
	return len(r.Evaluate(m)) > 0
}

// Escorts returns the escort roles this load triggers, de-duplicated by role so
// two thresholds firing for the same role do not double-count a vehicle.
func (r *Resolved) Escorts(m Measurements) []EscortRequirement {
	byRole := make(map[EscortRole]EscortRequirement, len(r.EscortThresholds))

	for _, threshold := range r.EscortThresholds {
		if threshold == nil {
			continue
		}

		actual := r.measurementFor(threshold.Trigger, m)
		if !threshold.Triggered(actual) {
			continue
		}

		existing, seen := byRole[threshold.Role]
		if seen && existing.Threshold >= threshold.ThresholdValue {
			continue
		}

		byRole[threshold.Role] = EscortRequirement{
			Role:      threshold.Role,
			Trigger:   threshold.Trigger,
			Threshold: threshold.ThresholdValue,
			Actual:    actual,
			Note:      threshold.Note,
		}
	}

	requirements := make([]EscortRequirement, 0, len(byRole))
	for _, requirement := range byRole {
		requirements = append(requirements, requirement)
	}

	sort.Slice(requirements, func(i, j int) bool {
		return requirements[i].Role < requirements[j].Role
	})

	return requirements
}

func (r *Resolved) measurementFor(trigger TriggerKind, m Measurements) float64 {
	switch trigger {
	case TriggerWidth:
		return m.WidthFeet
	case TriggerHeight:
		return m.HeightFeet
	case TriggerLength:
		return m.LengthFeet
	case TriggerWeight, TriggerSuperload:
		return float64(m.WeightPounds)
	default:
		return 0
	}
}

func (r *Resolved) EstimatedFee(m Measurements) decimal.Decimal {
	total := decimal.Zero

	if r.PermitBaseFee.Valid {
		total = total.Add(r.PermitBaseFee.Decimal)
	}

	if r.PermitPerMileFee.Valid && m.DistanceMiles > 0 {
		total = total.Add(
			r.PermitPerMileFee.Decimal.Mul(decimal.NewFromFloat(m.DistanceMiles)),
		)
	}

	return total
}

func (r *Resolved) Restrictions() []RestrictionKind {
	restrictions := make([]RestrictionKind, 0, 4)

	if r.DaylightOnly {
		restrictions = append(restrictions, RestrictionDaylightOnly)
	}
	if r.RushHourRestricted {
		restrictions = append(restrictions, RestrictionRushHour)
	}
	if r.WeekendRestricted {
		restrictions = append(restrictions, RestrictionWeekend)
	}
	if r.HolidayRestricted {
		restrictions = append(restrictions, RestrictionHoliday)
	}

	return restrictions
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
