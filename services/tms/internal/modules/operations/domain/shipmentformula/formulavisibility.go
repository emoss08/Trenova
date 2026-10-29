package shipmentformula

import "github.com/emoss08/trenova/internal/modules/operations/domain/shipment"

func Withhold(s *shipment.Shipment) *shipment.Shipment {
	if s == nil || !hasFormulaContent(s) {
		return s
	}

	redacted := *s
	redacted.FormulaTemplate = nil
	if redacted.RatingDetail != nil && redacted.RatingDetail.Trace != nil {
		detail := *redacted.RatingDetail
		detail.Trace = nil
		redacted.RatingDetail = &detail
	}

	return &redacted
}

func hasFormulaContent(s *shipment.Shipment) bool {
	return s.FormulaTemplate != nil ||
		(s.RatingDetail != nil && s.RatingDetail.Trace != nil)
}

func WithholdAll(items []*shipment.Shipment) []*shipment.Shipment {
	var redacted []*shipment.Shipment
	for i, item := range items {
		clean := Withhold(item)
		if clean == item && redacted == nil {
			continue
		}
		if redacted == nil {
			redacted = make([]*shipment.Shipment, len(items))
			copy(redacted, items[:i])
		}
		redacted[i] = clean
	}
	if redacted == nil {
		return items
	}

	return redacted
}
