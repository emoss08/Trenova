package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// TenderGuard is the narrow hook coverage mutations use to withdraw a live
// tender when its move gets covered or canceled outside the tender flow,
// without depending on the full tender service surface.
type TenderGuard interface {
	CancelLiveTenderForMove(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
		reason string,
	) error
	CancelLiveTendersForShipment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
		reason string,
	) error
}

// TenderResponseRequest is a carrier's answer to one offer, regardless of the
// channel it arrived on.
type TenderResponseRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	OfferID       pulid.ID              `json:"offerId"`
	Action        tender.ResponseAction `json:"action"`
	Source        tender.ResponseSource `json:"source"`
	DeclineReason string                `json:"declineReason"`
}

// TenderResponseRecorder is the single funnel every response channel (email
// token, EDI 990, dispatcher manual entry) resolves through. It never
// transitions offer state itself: it validates, then signals the tender's
// workflow, which owns every transition.
type TenderResponseRecorder interface {
	RecordResponse(ctx context.Context, req *TenderResponseRequest) error
}

// CarrierMoveAssigner is the seam the tender workflow uses to land an accepted
// offer on the carrier assignment spine without importing the assignment
// service package.
type CarrierMoveAssigner interface {
	AssignToMove(
		ctx context.Context,
		req *repositories.AssignMoveToCarrierRequest,
	) (*shipment.CarrierAssignment, error)
	ConfirmAssignment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		assignmentID pulid.ID,
	) error
}
