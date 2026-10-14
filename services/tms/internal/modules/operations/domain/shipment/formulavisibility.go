package shipment

func (s *Shipment) WithoutFormulaContent() *Shipment {
	if s == nil || !s.hasFormulaContent() {
		return s
	}

	redacted := *s
	redacted.FormulaTemplate = nil
	if redacted.RatingDetail != nil && redacted.RatingDetail.Expression != "" {
		detail := *redacted.RatingDetail
		detail.Expression = ""
		redacted.RatingDetail = &detail
	}

	return &redacted
}

func (s *Shipment) hasFormulaContent() bool {
	return s.FormulaTemplate != nil ||
		(s.RatingDetail != nil && s.RatingDetail.Expression != "")
}

func WithoutFormulaContent(items []*Shipment) []*Shipment {
	var redacted []*Shipment
	for i, item := range items {
		clean := item.WithoutFormulaContent()
		if clean == item && redacted == nil {
			continue
		}
		if redacted == nil {
			redacted = make([]*Shipment, len(items))
			copy(redacted, items[:i])
		}
		redacted[i] = clean
	}
	if redacted == nil {
		return items
	}

	return redacted
}
