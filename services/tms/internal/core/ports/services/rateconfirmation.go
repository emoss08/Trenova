package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// IssueTenderRateConfirmationRequest asks for the rate confirmation that
// documents an accepted tender offer. The carrier's acceptance IS the rate
// agreement, so the issued confirmation lands Confirmed with tender
// provenance.
type IssueTenderRateConfirmationRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ShipmentMoveID pulid.ID              `json:"shipmentMoveId"`
	OfferID        pulid.ID              `json:"offerId"`
	CarrierID      pulid.ID              `json:"carrierId"`
	Source         tender.ResponseSource `json:"source"`
	AcceptedAt     int64                 `json:"acceptedAt"`
}

// IssueTenderRateConfirmationResult reports what issuance produced.
// Issued=false with a Reason is a deterministic outcome (a dispatcher-managed
// agreement already stands, or the document could not be produced) that the
// caller surfaces to dispatch instead of retrying.
type IssueTenderRateConfirmationResult struct {
	RateConfirmationID pulid.ID `json:"rateConfirmationId"`
	Issued             bool     `json:"issued"`
	Reason             string   `json:"reason"`
}

// RateConfirmationIssuer is the seam the tender lifecycle uses to land the
// accepted offer's rate confirmation without importing the rate confirmation
// service package.
type RateConfirmationIssuer interface {
	IssueForTenderAcceptance(
		ctx context.Context,
		req *IssueTenderRateConfirmationRequest,
	) (*IssueTenderRateConfirmationResult, error)
}
