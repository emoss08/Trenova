package shipment

import (
	"github.com/emoss08/trenova/shared/pulid"
)

type RatingReviewApproval struct {
	Reviews      []string `json:"reviews"`
	ApprovedByID pulid.ID `json:"approvedById"`
	ApprovedAt   int64    `json:"approvedAt"`
}
