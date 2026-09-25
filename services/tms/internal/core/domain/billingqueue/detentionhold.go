package billingqueue

import (
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// DetentionHold is one detention charge on the item's shipment that is still
// waiting on an approver, and so is keeping the item from being approved.
type DetentionHold struct {
	OccurrenceID   pulid.ID                    `json:"occurrenceId"`
	StopID         pulid.ID                    `json:"stopId"`
	StopType       shipment.StopType           `json:"stopType"`
	LocationName   string                      `json:"locationName"`
	ClockStartAt   int64                       `json:"clockStartAt"`
	BillableAmount decimal.Decimal             `json:"billableAmount"`
	Currency       string                      `json:"currency"`
	Reason         detention.BillingHoldReason `json:"reason"`
}

// NewDetentionHolds describes the occurrences that hold billing, skipping any
// that do not.
func NewDetentionHolds(occurrences []*detention.DetentionOccurrence) []*DetentionHold {
	holds := make([]*DetentionHold, 0, len(occurrences))
	for _, occurrence := range occurrences {
		if occurrence == nil || !occurrence.HoldsBilling() {
			continue
		}
		holds = append(holds, &DetentionHold{
			OccurrenceID:   occurrence.ID,
			StopID:         occurrence.StopID,
			StopType:       occurrence.StopType,
			LocationName:   occurrence.LocationName,
			ClockStartAt:   occurrence.ClockStartAt,
			BillableAmount: occurrence.BillableAmount,
			Currency:       occurrence.Currency,
			Reason:         occurrence.BillingHoldReason(),
		})
	}

	return holds
}
