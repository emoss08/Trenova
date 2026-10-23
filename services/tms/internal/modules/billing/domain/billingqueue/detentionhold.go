package billingqueue

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
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

type RatingReviewHold struct {
	ShipmentID          pulid.ID `json:"shipmentId"`
	Reasons             []string `json:"reasons"`
	RatedAt             int64    `json:"ratedAt"`
	FormulaTemplateName string   `json:"formulaTemplateName"`
}

func (h *RatingReviewHold) Summary() string {
	return strings.Join(h.Reasons, "; ")
}

func HeldForRatingReview(item *BillingQueueItem) bool {
	return item != nil && item.ShipmentID.IsNotNil() &&
		item.BillType != BillTypeCreditMemo &&
		!IsTerminalStatus(item.Status)
}

func NewRatingReviewHold(item *BillingQueueItem) *RatingReviewHold {
	if !HeldForRatingReview(item) || item.Shipment == nil ||
		!item.Shipment.AwaitsRatingReview() {
		return nil
	}
	detail := item.Shipment.RatingDetail
	return &RatingReviewHold{
		ShipmentID:          item.Shipment.ID,
		Reasons:             detail.ReviewReasons(),
		RatedAt:             detail.RatedAt,
		FormulaTemplateName: detail.FormulaTemplateName,
	}
}
