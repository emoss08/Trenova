package inboundprocessing

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowersources"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// announce tells the desks a message was read.
//
// The event fires once the message is settled rather than when it arrived,
// because a desk asked to act on a message nobody has read yet has nothing to
// act on. It fires whatever the outcome: a desk that subscribes to intake is
// as interested in the tender it may take over as in the one it may not.
func (s *Service) announce(ctx context.Context, message *inboundmessage.InboundMessage) {
	related := make([]pulid.ID, 0, 3)
	for _, id := range []pulid.ID{
		message.MatchedShipmentID,
		message.MatchedCarrierID,
		message.MatchedCustomerID,
	} {
		if id.IsNotNil() {
			related = append(related, id)
		}
	}

	// The detail names the message, never what it says: a wait it ends hands
	// the line to the agent, which reads the message itself, and taints its
	// work when it does.
	agentports.PublishAgentEvent(ctx, s.events, agentports.AgentEvent{
		Kind:      agenttrigger.EventInboundMessageClassified,
		SubjectID: message.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: message.OrganizationID,
			BuID:  message.BusinessUnitID,
		},
		Related: related,
		Detail: "A message arrived from " + message.FromAddress + " (inbound message " +
			message.ID.String() + "). Read it with inbox_get_inbound_message before acting on it.",
	})
}

const (
	// EventNeedsReview is the notification a message raises when it lands in
	// front of a person. A quarantined one is not told this way: it is
	// critical on the watchtower, which already tells the same people.
	EventNeedsReview = "inbound_message.needs_review"
	// maxReviewRecipients bounds who is told about one message.
	maxReviewRecipients = 25
	// reviewNoticeWindow stops a message reprocessed after a redelivery or a
	// retry from telling everybody twice.
	reviewNoticeWindow = 24 * 60 * 60
	inboxNoticeSource  = "inbox"
)

// notifyNeedsReview tells the people who read the inbox that a message is
// waiting on them, in the words the watchtower uses for the same message.
func (s *Service) notifyNeedsReview(ctx context.Context, message *inboundmessage.InboundMessage) {
	if s.notifier == nil || message.Status != inboundmessage.StatusInReview {
		return
	}

	item := watchtowersources.DescribeInboundMessage(message)
	correlation := message.ID.String()
	now := timeutils.NowUnix()
	if _, err := s.notifier.NotifyPermitted(ctx, notificationservice.NotifyPermittedRequest{
		Tenant:      item.TenantInfo,
		Resource:    permission.ResourceInboundMessage,
		Operation:   permission.OpRead,
		Limit:       maxReviewRecipients,
		Now:         now,
		DedupeSince: now - reviewNoticeWindow,
		Notification: notification.Notification{
			EventType:     EventNeedsReview,
			Priority:      notification.PriorityMedium,
			Title:         item.Title,
			Message:       item.Summary,
			Source:        inboxNoticeSource,
			CorrelationID: &correlation,
			Data: map[string]any{
				"link":           item.Path,
				"messageId":      correlation,
				"classification": string(message.Classification),
			},
			RelatedEntities: map[string]any{"inboundMessageId": correlation},
		},
	}); err != nil {
		s.l.Warn("could not tell anyone a message needs review",
			zap.String("messageId", correlation), zap.Error(err))
	}
}
