package detention

import (
	"context"
	"sort"
	"strconv"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

// DetentionPolicyTier is one rung of a graduated rate ladder. Bands are
// expressed in minutes of billable detention measured from the end of free
// time, half-open as [FromMinute, ToMinute).
type DetentionPolicyTier struct {
	keel.Table `keel:"detention_policy_tiers"`

	ID                pulid.ID        `json:"id"                keel:"id,pk,autoid:dpt_"`
	BusinessUnitID    pulid.ID        `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID        `json:"organizationId"    keel:"organization_id,pk,tenant"`
	DetentionPolicyID pulid.ID        `json:"detentionPolicyId" keel:"detention_policy_id"`
	FromMinute        int32           `json:"fromMinute"        keel:"from_minute"`
	ToMinute          *int32          `json:"toMinute"          keel:"to_minute"`
	Rate              decimal.Decimal `json:"rate"              keel:"rate"`
	RateUnit          TierRateUnit    `json:"rateUnit"          keel:"rate_unit"`
	Label             string          `json:"label"             keel:"label,nullzero"`
	SortOrder         int32           `json:"sortOrder"         keel:"sort_order"`
	CreatedAt         int64           `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64           `json:"updatedAt"         keel:"updated_at,autoupdate"`
}

func (t *DetentionPolicyTier) BeforeUpdate(_ context.Context) error {
	if t.RateUnit == "" {
		t.RateUnit = TierRateUnitHour
	}

	return nil
}

func (t *DetentionPolicyTier) BeforeCreate(ctx context.Context) error {
	if t.RateUnit == "" {
		t.RateUnit = TierRateUnitHour
	}

	return nil
}

// Covers reports whether the tier owns the given billable minute.
func (t *DetentionPolicyTier) Covers(minute int32) bool {
	if minute < t.FromMinute {
		return false
	}
	if t.ToMinute != nil && minute >= *t.ToMinute {
		return false
	}
	return true
}

// SpanMinutes returns how many of the supplied billable minutes fall inside
// this tier, given the total billable duration.
func (t *DetentionPolicyTier) SpanMinutes(totalBillable int32) int32 {
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

// SortTiers orders tiers by their lower bound so the ladder walk is stable
// regardless of how the client submitted them.
func SortTiers(tiers []*DetentionPolicyTier) []*DetentionPolicyTier {
	sorted := make([]*DetentionPolicyTier, 0, len(tiers))
	for _, tier := range tiers {
		if tier != nil {
			sorted = append(sorted, tier)
		}
	}

	sort.SliceStable(sorted, func(a, b int) bool {
		return sorted[a].FromMinute < sorted[b].FromMinute
	})

	return sorted
}

// validateTiers enforces that the ladder starts at zero, never overlaps, and
// leaves no gap. A gap would silently bill nothing for a stretch of time, which
// is the kind of defect that only surfaces in a customer's rejection letter.
func validateTiers(tiers []*DetentionPolicyTier, multiErr *errortypes.MultiError) {
	if len(tiers) == 0 {
		return
	}

	type band struct {
		index int
		from  int32
		to    *int32
	}

	bands := make([]band, 0, len(tiers))

	for i, tier := range tiers {
		if tier == nil {
			continue
		}

		prefix := tierFieldPrefix(i)

		if tier.FromMinute < 0 {
			multiErr.Add(prefix+".fromMinute", errortypes.ErrInvalid,
				"Tier start cannot be negative")
			continue
		}

		if tier.ToMinute != nil && *tier.ToMinute <= tier.FromMinute {
			multiErr.Add(prefix+".toMinute", errortypes.ErrInvalid,
				"Tier end must be greater than tier start")
			continue
		}

		if tier.Rate.LessThan(decimal.Zero) {
			multiErr.Add(prefix+".rate", errortypes.ErrInvalid,
				"Tier rate cannot be negative")
			continue
		}

		if _, err := TierRateUnitFromString(string(tier.RateUnit)); err != nil {
			multiErr.Add(prefix+".rateUnit", errortypes.ErrInvalid,
				"Tier rate unit is invalid")
			continue
		}

		bands = append(bands, band{index: i, from: tier.FromMinute, to: tier.ToMinute})
	}

	if len(bands) == 0 {
		return
	}

	sort.Slice(bands, func(a, b int) bool { return bands[a].from < bands[b].from })

	if bands[0].from != 0 {
		multiErr.Add(tierFieldPrefix(bands[0].index)+".fromMinute", errortypes.ErrInvalid,
			"The first tier must start at minute 0 of billable detention")
	}

	openEnded := 0
	for _, b := range bands {
		if b.to == nil {
			openEnded++
		}
	}
	if openEnded > 1 {
		multiErr.Add("tiers", errortypes.ErrInvalid,
			"Only the final tier may be open-ended")
	}
	if openEnded == 1 && bands[len(bands)-1].to != nil {
		multiErr.Add("tiers", errortypes.ErrInvalid,
			"Only the final tier may be open-ended")
	}

	for i := 1; i < len(bands); i++ {
		prev := bands[i-1]
		curr := bands[i]
		prefix := tierFieldPrefix(curr.index)

		if prev.to == nil {
			multiErr.Add(prefix+".fromMinute", errortypes.ErrInvalid,
				"A tier cannot start after an open-ended tier")
			continue
		}

		if curr.from < *prev.to {
			multiErr.Add(prefix+".fromMinute", errortypes.ErrInvalid,
				"Tiers must not overlap")
			continue
		}

		if curr.from > *prev.to {
			multiErr.Add(prefix+".fromMinute", errortypes.ErrInvalid,
				"Tiers must be contiguous; this leaves an unbilled gap")
		}
	}
}

func tierFieldPrefix(index int) string {
	return "tiers[" + strconv.Itoa(index) + "]"
}

func (t *DetentionPolicyTier) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&t.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&t.DetentionPolicyID,
			validation.Required.Error("Detention policy is required"),
		),
		validation.Field(&t.FromMinute,
			validation.Min(int32(0)).Error("From minute cannot be negative"),
		),
		// A tier that ends before it starts covers no minutes at all, so the
		// charge it describes could never apply.
		validation.Field(&t.ToMinute,
			validation.Min(t.FromMinute+1).Error("To minute must be after from minute"),
		),
		validation.Field(&t.RateUnit,
			validation.Required.Error("Rate unit is required"),
			domainvalidation.ValidEnumFunc(TierRateUnitFromString, "Rate unit is invalid"),
		),
		validation.Field(&t.Label,
			validation.Length(0, maxTierLabelLength).
				Error("Label cannot be longer than 100 characters"),
		),
		validation.Field(&t.SortOrder,
			validation.Min(int32(0)).Error("Sort order cannot be negative"),
		),
	))

	if t.Rate.IsNegative() {
		multiErr.Add("rate", errortypes.ErrInvalid, "Rate cannot be negative")
	}
}

const maxTierLabelLength = 100
