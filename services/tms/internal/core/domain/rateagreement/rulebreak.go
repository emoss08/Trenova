package rateagreement

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RateAgreementRuleBreak)(nil)

// RateAgreementRuleBreak is one weight band within a rule.
//
// Bands are half open on their upper bound. A tariff that reads "1,000 to
// 2,000" means a two thousand pound shipment rates in the next band up, and
// closed intervals would let two bands both claim the boundary.
type RateAgreementRuleBreak struct {
	bun.BaseModel `bun:"table:rate_agreement_rule_breaks,alias:ragb" json:"-"`

	ID                  pulid.ID `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RateAgreementRuleID pulid.ID `json:"rateAgreementRuleId" bun:"rate_agreement_rule_id,type:VARCHAR(100),notnull"`

	FromWeight decimal.Decimal     `json:"fromWeight" bun:"from_weight,type:NUMERIC(12,2),notnull"`
	ToWeight   decimal.NullDecimal `json:"toWeight"   bun:"to_weight,type:NUMERIC(12,2),nullzero"`
	Rate       decimal.Decimal     `json:"rate"       bun:"rate,type:NUMERIC(19,6),notnull"`
	MinCharge  decimal.NullDecimal `json:"minCharge"  bun:"min_charge,type:NUMERIC(19,4),nullzero"`
	Label      string              `json:"label"      bun:"label,type:VARCHAR(50),nullzero"`
	SortOrder  int16               `json:"sortOrder"  bun:"sort_order,type:SMALLINT,notnull,default:0"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Rule *RateAgreementRule `json:"-" bun:"rel:belongs-to,join:rate_agreement_rule_id=id"`
}

// Contains reports whether a weight falls in this band.
func (ragb *RateAgreementRuleBreak) Contains(weight decimal.Decimal) bool {
	if weight.LessThan(ragb.FromWeight) {
		return false
	}

	if ragb.ToWeight.Valid && weight.GreaterThanOrEqual(ragb.ToWeight.Decimal) {
		return false
	}

	return true
}

// DisplayLabel is the label if one was given, and the band's own bounds
// otherwise, so a rating trace always has something to name the break by.
func (ragb *RateAgreementRuleBreak) DisplayLabel() string {
	if ragb.Label != "" {
		return ragb.Label
	}

	if ragb.ToWeight.Valid {
		return ragb.FromWeight.String() + "-" + ragb.ToWeight.Decimal.String()
	}

	return ragb.FromWeight.String() + "+"
}

func (ragb *RateAgreementRuleBreak) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(ragb,
		validation.Field(&ragb.Label,
			validation.Length(0, 50).Error("Label cannot be longer than 50 characters"),
		),
	))

	if ragb.FromWeight.IsNegative() {
		multiErr.Add("fromWeight", errortypes.ErrInvalid, "From weight cannot be negative")
	}

	if ragb.ToWeight.Valid && ragb.ToWeight.Decimal.LessThanOrEqual(ragb.FromWeight) {
		multiErr.Add(
			"toWeight",
			errortypes.ErrInvalid,
			"To weight must be greater than from weight",
		)
	}

	if ragb.Rate.IsNegative() {
		multiErr.Add("rate", errortypes.ErrInvalid, "Rate cannot be negative")
	}

	if ragb.MinCharge.Valid && ragb.MinCharge.Decimal.IsNegative() {
		multiErr.Add("minCharge", errortypes.ErrInvalid, "Minimum charge cannot be negative")
	}
}

// SortedBreaks returns a rule's bands in ascending weight order, skipping nils.
func SortedBreaks(breaks []*RateAgreementRuleBreak) []*RateAgreementRuleBreak {
	sorted := make([]*RateAgreementRuleBreak, 0, len(breaks))
	for _, ruleBreak := range breaks {
		if ruleBreak != nil {
			sorted = append(sorted, ruleBreak)
		}
	}

	sort.Slice(sorted, func(a, b int) bool {
		return sorted[a].FromWeight.LessThan(sorted[b].FromWeight)
	})

	return sorted
}

// validateBreaks rejects a band set that would leave a weight unpriced or
// priced twice. A gap means a shipment that cannot be rated; an overlap means
// the rate depends on row order.
func (rar *RateAgreementRule) validateBreaks(multiErr *errortypes.MultiError) {
	if len(rar.Breaks) == 0 {
		return
	}

	indexByBreak := make(map[*RateAgreementRuleBreak]int, len(rar.Breaks))
	for i, ruleBreak := range rar.Breaks {
		if ruleBreak == nil {
			continue
		}
		indexByBreak[ruleBreak] = i
		ruleBreak.Validate(multiErr.WithIndex("breaks", i))
	}

	sorted := SortedBreaks(rar.Breaks)
	if len(sorted) == 0 {
		return
	}

	if !sorted[0].FromWeight.IsZero() {
		multiErr.WithIndex("breaks", indexByBreak[sorted[0]]).Add(
			"fromWeight",
			errortypes.ErrInvalid,
			"The lightest break must start at zero",
		)
	}

	for i := 1; i < len(sorted); i++ {
		previous := sorted[i-1]
		current := sorted[i]
		currentErr := multiErr.WithIndex("breaks", indexByBreak[current])

		if !previous.ToWeight.Valid {
			currentErr.Add(
				"fromWeight",
				errortypes.ErrInvalid,
				"Only the heaviest break may be left open ended",
			)
			continue
		}

		if !previous.ToWeight.Decimal.Equal(current.FromWeight) {
			currentErr.Add(
				"fromWeight",
				errortypes.ErrInvalid,
				"Breaks must be contiguous with no gaps or overlaps",
			)
		}
	}

	if heaviest := sorted[len(sorted)-1]; heaviest.ToWeight.Valid {
		multiErr.WithIndex("breaks", indexByBreak[heaviest]).Add(
			"toWeight",
			errortypes.ErrInvalid,
			"The heaviest break must be open ended",
		)
	}
}

func (ragb *RateAgreementRuleBreak) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if ragb.ID.IsNil() {
			ragb.ID = pulid.MustNew("ragb_")
		}
		ragb.CreatedAt = now
		ragb.UpdatedAt = now
	case *bun.UpdateQuery:
		ragb.UpdatedAt = now
	}

	return nil
}

func (ragb *RateAgreementRuleBreak) GetID() pulid.ID {
	return ragb.ID
}

func (ragb *RateAgreementRuleBreak) GetOrganizationID() pulid.ID {
	return ragb.OrganizationID
}

func (ragb *RateAgreementRuleBreak) GetBusinessUnitID() pulid.ID {
	return ragb.BusinessUnitID
}

func (ragb *RateAgreementRuleBreak) GetTableName() string {
	return "rate_agreement_rule_breaks"
}
