package ratequoteports

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

// AuthorizeShopStrategy refuses to rank by margin for a caller without the
// margins field group: the order alone would tell them which carrier leaves
// the most.
func AuthorizeShopStrategy(strategy ShopStrategy, visible protectedfield.GroupSet) error {
	if strategy != ShopStrategyBestMargin || visible.Has(protectedfield.Margins) {
		return nil
	}

	return errortypes.NewAuthorizationError(
		"Ranking carriers by margin needs access to margins",
	)
}

// WithholdFields clears the margin an option leaves, and its note when the
// note is the margin verdict's explanation, which states the margin.
func (o *ShopOption) WithholdFields(visible protectedfield.GroupSet) {
	if o == nil || visible.Has(protectedfield.Margins) {
		return
	}
	if o.Margin != nil && o.Note == o.Margin.Explanation {
		o.Note = ""
	}
	o.Margin = nil
}

// WithholdFields clears the sell total the options were measured against:
// beside each option's cost it gives that option's margin away.
func (r *ShopResult) WithholdFields(visible protectedfield.GroupSet) {
	if r == nil || visible.Has(protectedfield.Margins) {
		return
	}
	r.SellTotal = decimal.NullDecimal{}
	for _, option := range r.Options {
		option.WithholdFields(visible)
	}
}
