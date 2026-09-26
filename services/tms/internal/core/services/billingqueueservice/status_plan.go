package billingqueueservice

import (
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// PlanStatusChange is what UpdateStatus does to the item, refused where the
// actor may not make the move.
func PlanStatusChange(
	entity *billingqueue.BillingQueueItem,
	req *services.UpdateBillingQueueStatusRequest,
	actor *services.RequestActor,
	now int64,
) error {
	if err := guardAgentTransition(actor, req.NewStatus); err != nil {
		return err
	}

	return PlanTransition(entity, req, actor, now)
}

// PlanTransition is the move itself, whoever makes it. A preview of a
// decision only a person makes plans it for the person who will approve it;
// the write still goes through UpdateStatus, which refuses an agent.
func PlanTransition(
	entity *billingqueue.BillingQueueItem,
	req *services.UpdateBillingQueueStatusRequest,
	actor *services.RequestActor,
	now int64,
) error {
	if err := checkStatusTransition(entity, req); err != nil {
		return err
	}

	entity.Status = req.NewStatus
	applyStatusFields(entity, req, actor, now)

	return nil
}

// AgentMayMoveTo reports whether an agent principal may move an item to a
// status. Review, hold, exception and a return to operations create no money
// and are undone by moving the item again. Approving creates the invoice,
// canceling drops the charge, posting books it and returning an item to
// review re-opens what a person closed, so those stay a person's.
func AgentMayMoveTo(status billingqueue.Status) bool {
	switch status { //nolint:exhaustive // every other status is a person's decision
	case billingqueue.StatusInReview,
		billingqueue.StatusOnHold,
		billingqueue.StatusException,
		billingqueue.StatusSentBackToOps:
		return true
	default:
		return false
	}
}

func guardAgentTransition(actor *services.RequestActor, status billingqueue.Status) error {
	if actor.IsAgent() && !AgentMayMoveTo(status) {
		return errAgentCannotTransition()
	}

	return nil
}

// SendBackComment is the operations note a return to operations posts on the
// item's shipment.
func SendBackComment(
	entity *billingqueue.BillingQueueItem,
	userID pulid.ID,
) *shipment.ShipmentComment {
	comment := "Sent back from billing"
	if entity.ExceptionReasonCode != nil {
		comment += ": " + string(*entity.ExceptionReasonCode)
	}
	if entity.ExceptionNotes != "" {
		comment += "\n\n" + entity.ExceptionNotes
	}

	return &shipment.ShipmentComment{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ShipmentID:     entity.ShipmentID,
		UserID:         userID,
		Comment:        comment,
		Type:           shipment.CommentTypeBilling,
		Visibility:     shipment.CommentVisibilityOperations,
		Priority:       shipment.CommentPriorityHigh,
		Source:         shipment.CommentSourceSystem,
	}
}

func checkStatusTransition(
	entity *billingqueue.BillingQueueItem,
	req *services.UpdateBillingQueueStatusRequest,
) error {
	if billingqueue.IsAllowedTransition(entity.Status, req.NewStatus) {
		return nil
	}

	return errortypes.NewValidationError(
		"status",
		errortypes.ErrInvalidOperation,
		"Cannot transition from {0} to {1}", string(entity.Status), string(req.NewStatus),
	)
}

func errAgentCannotTransition() error {
	return errortypes.NewValidationError(
		"actor",
		errortypes.ErrForbidden,
		"Agent principals cannot transition billing queue items; a human must decide",
	)
}

func applyStatusFields(
	entity *billingqueue.BillingQueueItem,
	req *services.UpdateBillingQueueStatusRequest,
	actor *services.RequestActor,
	now int64,
) {
	switch req.NewStatus {
	case billingqueue.StatusInReview:
		if entity.ReviewStartedAt == nil {
			entity.ReviewStartedAt = &now
		}
		entity.ReviewCompletedAt = nil
	case billingqueue.StatusApproved:
		entity.ReviewCompletedAt = &now
		if req.ReviewNotes != "" {
			entity.ReviewNotes = req.ReviewNotes
		}
	case billingqueue.StatusPosted:
		entity.ReviewCompletedAt = &now
	case billingqueue.StatusSentBackToOps, billingqueue.StatusException:
		entity.ExceptionReasonCode = req.ExceptionReasonCode
		entity.ExceptionNotes = req.ExceptionNotes
	case billingqueue.StatusCanceled:
		userID := actor.UserID
		entity.CanceledByID = &userID
		entity.CanceledAt = &now
		entity.CancelReason = req.CancelReason
	case billingqueue.StatusReadyForReview:
		entity.ExceptionReasonCode = nil
		entity.ExceptionNotes = ""
	case billingqueue.StatusOnHold:
		if req.ReviewNotes != "" {
			entity.ReviewNotes = req.ReviewNotes
		}
	}
}

// PlanAssignBiller is what AssignBiller does to the item: it names the biller,
// and an item waiting for review moves into review with them.
func PlanAssignBiller(entity *billingqueue.BillingQueueItem, billerID pulid.ID, now int64) error {
	if billingqueue.IsTerminalStatus(entity.Status) {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Cannot assign a biller to a billing queue item in {0} status", string(entity.Status),
		)
	}

	entity.AssignedBillerID = &billerID
	if entity.Status == billingqueue.StatusReadyForReview {
		entity.Status = billingqueue.StatusInReview
		entity.ReviewStartedAt = &now
	}

	return nil
}
