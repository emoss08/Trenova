package inboundmessageservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/pkg/pagination"
)

// project puts a message on the watchtower, or takes it off.
//
// Only a message waiting on a person belongs there. One the desk answered by
// itself is work that happened rather than work owed, and leaving it on the
// feed would bury the ones that are actually waiting — so settling a message
// resolves its item as readily as it raises one.
func (s *Service) project(ctx context.Context, message *inboundmessage.InboundMessage) {
	if s.watchtower == nil {
		return
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: message.OrganizationID,
		BuID:  message.BusinessUnitID,
	}

	if !message.NeedsReview() {
		s.watchtower.Resolve(
			ctx, tenantInfo, watchtower.SourceInboundMessage, message.ID.String(),
		)

		return
	}

	s.watchtower.Upsert(ctx, watchtowersources.DescribeInboundMessage(message))
}

// announce tells the desks a message was read.
//
// The event fires once the message is settled rather than when it arrived,
// because a desk asked to act on a message nobody has read yet has nothing to
// act on. It fires whatever the outcome: a desk that subscribes to intake is
// as interested in the tender it may take over as in the one it may not.
func (s *Service) announce(ctx context.Context, message *inboundmessage.InboundMessage) {
	services.PublishAgentEvent(ctx, s.events, services.AgentEvent{
		Kind:      agent.EventInboundMessageClassified,
		SubjectID: message.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: message.OrganizationID,
			BuID:  message.BusinessUnitID,
		},
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
	reviewNoticeWindow  = 24 * 60 * 60
	inboxNoticeSource   = "inbox"
	inboxRealtimeAction = "updated"
)

// publish tells an open inbox that a message changed, so a lane, a count or a
// reading pane moves without a reload.
func (s *Service) publish(ctx context.Context, orgID, buID, messageID pulid.ID, action string) {
	if s.realtime == nil {
		return
	}

	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ActorType:      services.PrincipalTypeSystem,
		Resource:       permission.ResourceInboundMessage.String(),
		Action:         action,
		RecordID:       messageID,
	}); err != nil {
		s.l.Warn("inbox invalidation lost",
			zap.String("messageId", messageID.String()), zap.Error(err))
	}
}

func (s *Service) publishMessage(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	action string,
) {
	s.publish(ctx, message.OrganizationID, message.BusinessUnitID, message.ID, action)
}

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
