package billingqueueservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type systemEventsKey struct{}

// systemEvents marks what follows as the system's own doing: a transfer, a
// default biller from the customer's profile. The request still carries the
// person whose action set it off, but the activity should not say they did it.
func systemEvents(ctx context.Context) context.Context {
	return context.WithValue(ctx, systemEventsKey{}, true)
}

func isSystemEvent(ctx context.Context) bool {
	v, _ := ctx.Value(systemEventsKey{}).(bool)
	return v
}

// newEvent is one activity line attributed to whoever acted: the system when
// nobody did or the system acted on their behalf, an agent, or the person.
func (s *service) newEvent(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	kind billingqueue.EventKind,
	text string,
	actor *services.RequestActor,
	payload map[string]any,
) *billingqueue.ItemEvent {
	event := &billingqueue.ItemEvent{
		OrganizationID: item.OrganizationID,
		BusinessUnitID: item.BusinessUnitID,
		ItemID:         item.ID,
		Kind:           kind,
		Text:           text,
		ActorType:      billingqueue.EventActorSystem,
		Payload:        payload,
	}
	switch {
	case actor == nil || isSystemEvent(ctx):
	case actor.IsAgent():
		event.ActorType = billingqueue.EventActorAgent
		event.ActorName = "Agent"
		if id := actor.PrincipalID; id.IsNotNil() {
			event.ActorID = &id
		}
	case actor.UserID.IsNotNil():
		event.ActorType = billingqueue.EventActorUser
		id := actor.UserID
		event.ActorID = &id
		event.ActorName = s.personName(ctx, actor)
	}

	return event
}

func (s *service) personName(ctx context.Context, actor *services.RequestActor) string {
	if s.userRepo == nil {
		return ""
	}
	user, err := s.userRepo.GetByID(ctx, repositories.GetUserByIDRequest{
		LookupUserID: actor.UserID,
		TenantInfo: pagination.TenantInfo{
			OrgID: actor.OrganizationID,
			BuID:  actor.BusinessUnitID,
		},
	})
	if err != nil || user == nil {
		return ""
	}

	return user.Name
}

// recordEvent writes one activity line. Activity is a record of what already
// happened, so failing to write it never undoes the action itself.
func (s *service) recordEvent(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	kind billingqueue.EventKind,
	text string,
	actor *services.RequestActor,
	payload map[string]any,
) {
	if s.reviewRepo == nil || item == nil {
		return
	}
	if err := s.reviewRepo.CreateEvents(ctx, s.newEvent(ctx, item, kind, text, actor, payload)); err != nil {
		s.l.Warn("failed to record billing queue activity",
			zap.String("billingQueueItemId", item.ID.String()),
			zap.String("kind", string(kind)),
			zap.Error(err),
		)
	}
}

// recordStatusEvent says in words what a status change did.
func (s *service) recordStatusEvent(
	ctx context.Context,
	previous, updated *billingqueue.BillingQueueItem,
	created *services.CreateInvoiceFromBillingQueueResult,
	actor *services.RequestActor,
) {
	if previous == nil || updated == nil || previous.Status == updated.Status {
		return
	}

	switch updated.Status { //nolint:exhaustive // the rest read as a plain move
	case billingqueue.StatusApproved:
		text := "Approved the invoice for " + billingqueue.FormatMoney(updated.AllocatedTotalAmount)
		if created != nil && created.Invoice == nil {
			text = "Approved onto the customer's statement"
		}
		s.recordEvent(ctx, updated, billingqueue.EventApproved, text, actor, nil)
	case billingqueue.StatusOnHold:
		text := "Put on hold"
		if updated.HoldReasonCode != nil {
			text += " · " + updated.HoldReasonCode.Phrase()
		}
		s.recordEvent(ctx, updated, billingqueue.EventHeld, text, actor, nil)
	case billingqueue.StatusPosted:
		// Posting from the item writes its own line with the invoice number.
	default:
		if previous.Status == billingqueue.StatusOnHold {
			s.recordEvent(ctx, updated, billingqueue.EventReleased, "Released the hold", actor, nil)
			return
		}
		if previous.Status == billingqueue.StatusReadyForReview &&
			updated.Status == billingqueue.StatusInReview {
			return
		}
		s.recordEvent(ctx, updated, billingqueue.EventStatusChanged,
			"Moved to "+statusWords(updated.Status), actor, nil)
	}
}

func statusWords(status billingqueue.Status) string {
	switch status {
	case billingqueue.StatusReadyForReview:
		return "ready for review"
	case billingqueue.StatusInReview:
		return "in review"
	case billingqueue.StatusSentBackToOps:
		return "sent back to operations"
	default:
		return strings.ToLower(string(status))
	}
}

func (s *service) ListActivity(
	ctx context.Context,
	req *services.ListBillingQueueActivityRequest,
) (*repositories.BillingQueueEventPage, error) {
	if s.reviewRepo == nil {
		return nil, errortypes.NewBusinessError("Billing queue activity is unavailable")
	}
	if _, err := s.repo.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	return s.reviewRepo.ListEvents(ctx, &repositories.ListBillingQueueEventsRequest{
		TenantInfo: req.TenantInfo,
		ItemID:     req.ItemID,
		Limit:      req.Limit,
		BeforeAt:   req.BeforeAt,
		BeforeID:   req.BeforeID,
	})
}

func (s *service) Neighbors(
	ctx context.Context,
	req *repositories.GetBillingQueueNeighborsRequest,
) (*repositories.BillingQueueNeighbors, error) {
	if s.reviewRepo == nil {
		return nil, errortypes.NewBusinessError("The billing queue's order is unavailable")
	}

	return s.reviewRepo.GetNeighbors(ctx, req)
}

// maxSummaries bounds a summaries read to a page of the queue table.
const maxSummaries = 200

func (s *service) Summaries(
	ctx context.Context,
	req *repositories.ListBillingQueueSummariesRequest,
) ([]*repositories.BillingQueueItemSummary, error) {
	if s.reviewRepo == nil {
		return nil, errortypes.NewBusinessError("Billing queue summaries are unavailable")
	}
	if len(req.ItemIDs) > maxSummaries {
		return nil, errortypes.NewValidationError(
			"ids", errortypes.ErrInvalid, "Ask for at most 200 items at a time",
		)
	}

	return s.reviewRepo.ListSummaries(ctx, req)
}
