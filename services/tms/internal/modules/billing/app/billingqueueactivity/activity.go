package billingqueueactivity

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueueevent"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuereview"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"go.uber.org/zap"
)

type Params struct {
	Logger     *zap.Logger
	ReviewRepo billingqueueports.BillingQueueReviewRepository
	UserRepo   tenantports.UserRepository
	Realtime   realtimeports.RealtimeService
}

// Recorder keeps a billing queue item's activity: the events on its
// timeline and the invalidations that refresh it on screen.
type Recorder struct {
	l          *zap.Logger
	reviewRepo billingqueueports.BillingQueueReviewRepository
	userRepo   tenantports.UserRepository
	realtime   realtimeports.RealtimeService
}

func New(p Params) *Recorder {
	return &Recorder{
		l:          p.Logger,
		reviewRepo: p.ReviewRepo,
		userRepo:   p.UserRepo,
		realtime:   p.Realtime,
	}
}

type systemEventsKey struct{}

// systemEvents marks what follows as the system's own doing: a transfer, a
// default biller from the customer's profile. The request still carries the
// person whose action set it off, but the activity should not say they did it.
func SystemEvents(ctx context.Context) context.Context {
	return context.WithValue(ctx, systemEventsKey{}, true)
}

func isSystemEvent(ctx context.Context) bool {
	v, _ := ctx.Value(systemEventsKey{}).(bool)
	return v
}

// newEvent is one activity line attributed to whoever acted: the system when
// nobody did or the system acted on their behalf, an agent, or the person.
func (s *Recorder) NewEvent(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	kind billingqueueevent.EventKind,
	text string,
	actor *permissionports.RequestActor,
	payload map[string]any,
) *billingqueueevent.ItemEvent {
	event := &billingqueueevent.ItemEvent{
		OrganizationID: item.OrganizationID,
		BusinessUnitID: item.BusinessUnitID,
		ItemID:         item.ID,
		Kind:           kind,
		Text:           text,
		ActorType:      billingqueueevent.EventActorSystem,
		Payload:        payload,
	}
	switch {
	case actor == nil || isSystemEvent(ctx):
	case actor.IsAgent():
		event.ActorType = billingqueueevent.EventActorAgent
		event.ActorName = "Agent"
		if id := actor.PrincipalID; id.IsNotNil() {
			event.ActorID = &id
		}
	case actor.UserID.IsNotNil():
		event.ActorType = billingqueueevent.EventActorUser
		id := actor.UserID
		event.ActorID = &id
		event.ActorName = s.personName(ctx, actor)
	}

	return event
}

func (s *Recorder) personName(ctx context.Context, actor *permissionports.RequestActor) string {
	if s.userRepo == nil {
		return ""
	}
	user, err := s.userRepo.GetByID(ctx, tenantports.GetUserByIDRequest{
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
func (s *Recorder) RecordEvent(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	kind billingqueueevent.EventKind,
	text string,
	actor *permissionports.RequestActor,
	payload map[string]any,
) {
	if s.reviewRepo == nil || item == nil {
		return
	}
	if err := s.reviewRepo.CreateEvents(
		ctx,
		s.NewEvent(ctx, item, kind, text, actor, payload),
	); err != nil {
		s.l.Warn("failed to record billing queue activity",
			zap.String("billingQueueItemId", item.ID.String()),
			zap.String("kind", string(kind)),
			zap.Error(err),
		)
	}
}

// recordStatusEvent says in words what a status change did.
func (s *Recorder) RecordStatusEvent(
	ctx context.Context,
	previous, updated *billingqueue.BillingQueueItem,
	created *invoiceports.CreateInvoiceFromBillingQueueResult,
	actor *permissionports.RequestActor,
) {
	if previous == nil || updated == nil || previous.Status == updated.Status {
		return
	}

	switch updated.Status { //nolint:exhaustive // the rest read as a plain move
	case billingqueue.StatusApproved:
		text := "Approved the invoice for " + billingqueuereview.FormatMoney(
			updated.AllocatedTotalAmount,
		)
		if created != nil && created.Invoice == nil {
			text = "Approved onto the customer's statement"
		}
		s.RecordEvent(ctx, updated, billingqueueevent.EventApproved, text, actor, nil)
	case billingqueue.StatusOnHold:
		text := "Put on hold"
		if updated.HoldReasonCode != nil {
			text += " · " + updated.HoldReasonCode.Phrase()
		}
		s.RecordEvent(ctx, updated, billingqueueevent.EventHeld, text, actor, nil)
	case billingqueue.StatusPosted:
		// Posting from the item writes its own line with the invoice number.
	default:
		if previous.Status == billingqueue.StatusOnHold {
			s.RecordEvent(
				ctx,
				updated,
				billingqueueevent.EventReleased,
				"Released the hold",
				actor,
				nil,
			)
			return
		}
		if previous.Status == billingqueue.StatusReadyForReview &&
			updated.Status == billingqueue.StatusInReview {
			return
		}
		s.RecordEvent(ctx, updated, billingqueueevent.EventStatusChanged,
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
	case billingqueue.StatusApproved,
		billingqueue.StatusPosted,
		billingqueue.StatusOnHold,
		billingqueue.StatusException,
		billingqueue.StatusCanceled:
	}

	return strings.ToLower(string(status))
}

func (s *Recorder) PublishInvalidation(
	ctx context.Context,
	entity *billingqueue.BillingQueueItem,
	actor kernelports.AuditActor,
	action string,
	payload any,
) {
	err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       permission.ResourceBillingQueue.String(),
		Action:         action,
		RecordID:       entity.ID,
		Entity:         payload,
	})
	if err != nil {
		s.l.Warn("failed to publish billing queue invalidation", zap.Error(err))
	}
}
