package detention

import (
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// TierSnapshot is one frozen rung of the rate ladder.
type TierSnapshot struct {
	FromMinute int32           `json:"fromMinute"`
	ToMinute   *int32          `json:"toMinute"`
	Rate       decimal.Decimal `json:"rate"`
	RateUnit   TierRateUnit    `json:"rateUnit"`
	Label      string          `json:"label"`
}

// PolicySnapshot freezes the contract terms that governed an occurrence at the
// moment it was computed. Detention is disputed months after the fact and the
// policy may have been renegotiated since; replaying the live policy would
// answer the wrong question. Everything needed to recompute the charge and to
// explain it to a customer lives here.
type PolicySnapshot struct {
	PolicyID                    pulid.ID                `json:"policyId"`
	PolicyCode                  string                  `json:"policyCode"`
	PolicyName                  string                  `json:"policyName"`
	PolicyVersion               int64                   `json:"policyVersion"`
	SpecificityScore            int32                   `json:"specificityScore"`
	Priority                    int16                   `json:"priority"`
	ClockStartBasis             ClockStartBasis         `json:"clockStartBasis"`
	LateArrivalRule             LateArrivalRule         `json:"lateArrivalRule"`
	LateArrivalGraceMinutes     int16                   `json:"lateArrivalGraceMinutes"`
	FreeMinutes                 int32                   `json:"freeMinutes"`
	PayFreeMinutes              int32                   `json:"payFreeMinutes"`
	MinimumBillableMinutes      int32                   `json:"minimumBillableMinutes"`
	BillingIncrementMinutes     int16                   `json:"billingIncrementMinutes"`
	RoundingMode                RoundingMode            `json:"roundingMode"`
	RateSource                  RateSource              `json:"rateSource"`
	AccessorialChargeID         pulid.ID                `json:"accessorialChargeId"`
	AccessorialCode             string                  `json:"accessorialCode"`
	FlatRate                    decimal.Decimal         `json:"flatRate"`
	FlatRateUnit                TierRateUnit            `json:"flatRateUnit"`
	Tiers                       []TierSnapshot          `json:"tiers,omitempty"`
	MaxBillableMinutesPerStop   *int32                  `json:"maxBillableMinutesPerStop"`
	MaxChargePerStop            decimal.NullDecimal     `json:"maxChargePerStop"`
	MaxChargePerDay             decimal.NullDecimal     `json:"maxChargePerDay"`
	MaxChargePerShipment        decimal.NullDecimal     `json:"maxChargePerShipment"`
	DayBoundaryMode             CapScope                `json:"dayBoundaryMode"`
	ConvertToLayoverAtMinutes   *int32                  `json:"convertToLayoverAtMinutes"`
	LayoverAccessorialChargeID  *pulid.ID               `json:"layoverAccessorialChargeId"`
	NotificationRequirement     NotificationRequirement `json:"notificationRequirement"`
	NotificationLeadMinutes     int16                   `json:"notificationLeadMinutes"`
	NotificationDeadlineMinutes int16                   `json:"notificationDeadlineMinutes"`
	UnnotifiedBehavior          UnnotifiedBehavior      `json:"unnotifiedBehavior"`
	RequireApprovalOverAmount   decimal.NullDecimal     `json:"requireApprovalOverAmount"`
	AutoApproveUnderAmount      decimal.NullDecimal     `json:"autoApproveUnderAmount"`
	Currency                    string                  `json:"currency"`
	CapturedAt                  int64                   `json:"capturedAt"`
}

// SnapshotInput carries the resolved context a snapshot needs beyond the policy
// itself.
type SnapshotInput struct {
	Policy          *DetentionPolicy
	StopType        string
	FreeMinutes     int32
	AccessorialCode string
	FlatRate        decimal.Decimal
	FlatRateUnit    TierRateUnit
	CapturedAt      int64
}

// NewPolicySnapshot freezes a policy into an immutable record.
func NewPolicySnapshot(in SnapshotInput) PolicySnapshot {
	p := in.Policy
	if p == nil {
		return PolicySnapshot{CapturedAt: in.CapturedAt}
	}

	snapshot := PolicySnapshot{
		PolicyID:                    p.ID,
		PolicyCode:                  p.Code,
		PolicyName:                  p.Name,
		PolicyVersion:               p.Version,
		SpecificityScore:            p.SpecificityScore,
		Priority:                    p.Priority,
		ClockStartBasis:             p.ClockStartBasis,
		LateArrivalRule:             p.LateArrivalRule,
		LateArrivalGraceMinutes:     p.LateArrivalGraceMinutes,
		FreeMinutes:                 in.FreeMinutes,
		PayFreeMinutes:              p.PayFreeMinutesOrDefault(),
		MinimumBillableMinutes:      p.MinimumBillableMinutes,
		BillingIncrementMinutes:     p.BillingIncrementMinutes,
		RoundingMode:                p.RoundingMode,
		RateSource:                  p.RateSource,
		AccessorialChargeID:         p.AccessorialChargeID,
		AccessorialCode:             in.AccessorialCode,
		FlatRate:                    in.FlatRate,
		FlatRateUnit:                in.FlatRateUnit,
		MaxBillableMinutesPerStop:   p.MaxBillableMinutesPerStop,
		MaxChargePerStop:            p.MaxChargePerStop,
		MaxChargePerDay:             p.MaxChargePerDay,
		MaxChargePerShipment:        p.MaxChargePerShipment,
		DayBoundaryMode:             p.DayBoundaryMode,
		ConvertToLayoverAtMinutes:   p.ConvertToLayoverAtMinutes,
		LayoverAccessorialChargeID:  p.LayoverAccessorialChargeID,
		NotificationRequirement:     p.NotificationRequirement,
		NotificationLeadMinutes:     p.NotificationLeadMinutes,
		NotificationDeadlineMinutes: p.NotificationDeadlineMinutes,
		UnnotifiedBehavior:          p.UnnotifiedBehavior,
		RequireApprovalOverAmount:   p.RequireApprovalOverAmount,
		AutoApproveUnderAmount:      p.AutoApproveUnderAmount,
		Currency:                    p.Currency,
		CapturedAt:                  in.CapturedAt,
	}

	if p.RateSource == RateSourceTiers {
		sorted := SortTiers(p.Tiers)
		snapshot.Tiers = make([]TierSnapshot, 0, len(sorted))
		for _, tier := range sorted {
			snapshot.Tiers = append(snapshot.Tiers, TierSnapshot{
				FromMinute: tier.FromMinute,
				ToMinute:   tier.ToMinute,
				Rate:       tier.Rate,
				RateUnit:   tier.RateUnit,
				Label:      tier.Label,
			})
		}
	}

	return snapshot
}

// SpanMinutes returns how many of the supplied billable minutes fall inside
// this frozen tier.
func (t TierSnapshot) SpanMinutes(totalBillable int32) int32 {
	if totalBillable <= t.FromMinute {
		return 0
	}

	upper := totalBillable
	if t.ToMinute != nil && *t.ToMinute < upper {
		upper = *t.ToMinute
	}

	span := upper - t.FromMinute
	if span < 0 {
		return 0
	}

	return span
}
