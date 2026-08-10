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

// TenderPlanOffer is one offer's position in the workflow's execution plan.
type TenderPlanOffer struct {
	OfferID         pulid.ID           `json:"offerId"`
	Rank            int16              `json:"rank"`
	Status          tender.OfferStatus `json:"status"`
	OfferTTLSeconds int32              `json:"offerTtlSeconds"`
	ExpiresAt       *int64             `json:"expiresAt"`
}

// TenderPlan is the workflow's view of a tender: ordered offers with their
// current statuses, so a restarted workflow resumes exactly where the rows
// say it left off.
type TenderPlan struct {
	TenderID pulid.ID          `json:"tenderId"`
	Status   tender.Status     `json:"status"`
	Mode     tender.Mode       `json:"mode"`
	Offers   []TenderPlanOffer `json:"offers"`
}

// TenderDispatchResult reports one offer's channel dispatch. Delivered=false
// means the offer was marked DeliveryFailed and the plan should move on — it
// is an expected business outcome, not an activity error.
type TenderDispatchResult struct {
	Delivered bool   `json:"delivered"`
	ExpiresAt int64  `json:"expiresAt"`
	Reason    string `json:"reason"`
}

// TenderAcceptResult reports what an acceptance produced. Ignored means the
// offer or tender had already moved on and nothing changed; NeedsReview means
// the acceptance stuck but the assignment could not be created.
type TenderAcceptResult struct {
	Ignored      bool   `json:"ignored"`
	NeedsReview  bool   `json:"needsReview"`
	ReviewReason string `json:"reviewReason"`
}

// TenderLifecycle is the surface the tender workflow's activities drive. Every
// method is idempotent through compare-and-set transitions, so a retried
// activity never double-applies.
type TenderLifecycle interface {
	LoadPlan(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		tenderID pulid.ID,
	) (*TenderPlan, error)
	DispatchOffer(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		offerID pulid.ID,
	) (*TenderDispatchResult, error)
	ExpireOffer(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		offerID pulid.ID,
	) error
	DeclineOffer(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		offerID pulid.ID,
		source tender.ResponseSource,
		reason string,
	) error
	AcceptOffer(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		offerID pulid.ID,
		source tender.ResponseSource,
	) (*TenderAcceptResult, error)
	FinalizeAccepted(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		tenderID pulid.ID,
		acceptedOfferID pulid.ID,
	) error
	MarkNeedsReview(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		tenderID pulid.ID,
		offerID pulid.ID,
		reason string,
	) error
	MarkExhausted(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		tenderID pulid.ID,
	) error
	CancelFromWorkflow(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		tenderID pulid.ID,
		reason string,
		actorUserID pulid.ID,
	) error
}

// SendTenderOfferEDIRequest asks the EDI layer to deliver one offer as an
// outbound 204 to the carrier's linked partner.
type SendTenderOfferEDIRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	OfferID    pulid.ID              `json:"offerId"`
}

type SendTenderOfferEDIResult struct {
	MessageID pulid.ID `json:"messageId"`
}

// EDITenderChannel is the EDI delivery seam for tender offers, implemented by
// the EDI service so the tender service never touches EDI internals.
type EDITenderChannel interface {
	SendTenderOffer(
		ctx context.Context,
		req *SendTenderOfferEDIRequest,
	) (*SendTenderOfferEDIResult, error)
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
