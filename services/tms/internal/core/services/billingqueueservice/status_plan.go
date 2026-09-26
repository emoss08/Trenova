package billingqueueservice

import (
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func PlanStatusChange(
	entity *billingqueue.BillingQueueItem,
	req *services.UpdateBillingQueueStatusRequest,
	actor *services.RequestActor,
	now int64,
) error {
	if actor.IsAgent() {
		return errAgentCannotTransition()
	}
	if err := checkStatusTransition(entity, req); err != nil {
		return err
	}

	entity.Status = req.NewStatus
	applyStatusFields(entity, req, actor, now)

	return nil
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
