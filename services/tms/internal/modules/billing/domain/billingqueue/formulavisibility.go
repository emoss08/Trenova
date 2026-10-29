package billingqueue

import "github.com/emoss08/trenova/internal/modules/operations/domain/shipmentformula"

func (i *BillingQueueItem) WithoutFormulaContent() *BillingQueueItem {
	if i == nil || i.Shipment == nil {
		return i
	}

	shp := shipmentformula.Withhold(i.Shipment)
	if shp == i.Shipment {
		return i
	}

	redacted := *i
	redacted.Shipment = shp

	return &redacted
}

func WithoutFormulaContent(items []*BillingQueueItem) []*BillingQueueItem {
	var redacted []*BillingQueueItem
	for i, item := range items {
		clean := item.WithoutFormulaContent()
		if clean == item && redacted == nil {
			continue
		}
		if redacted == nil {
			redacted = make([]*BillingQueueItem, len(items))
			copy(redacted, items[:i])
		}
		redacted[i] = clean
	}
	if redacted == nil {
		return items
	}

	return redacted
}
