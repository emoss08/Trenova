package ratequoteports

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

// AuthorizeShopStrategy refuses to rank by margin for a caller without the
// margins field group: the order alone would tell them which carrier leaves
// the most.
func AuthorizeShopStrategy(strategy ShopStrategy, visible permission.FieldGroupSet) error {
	if strategy != ShopStrategyBestMargin || visible.Has(permission.FieldGroupMargins) {
		return nil
	}

	return errortypes.NewAuthorizationError(
		"Ranking carriers by margin needs access to margins",
	)
}

// WithholdFields clears the margin an option leaves, and its note when the
// note is the margin verdict's explanation, which states the margin.
func (o *ShopOption) WithholdFields(visible permission.FieldGroupSet) {
	if o == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	if o.Margin != nil && o.Note == o.Margin.Explanation {
		o.Note = ""
	}
	o.Margin = nil
}

// WithholdFields clears the sell total the options were measured against:
// beside each option's cost it gives that option's margin away.
func (r *ShopResult) WithholdFields(visible permission.FieldGroupSet) {
	if r == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	r.SellTotal = decimal.NullDecimal{}
	for _, option := range r.Options {
		option.WithholdFields(visible)
	}
}
