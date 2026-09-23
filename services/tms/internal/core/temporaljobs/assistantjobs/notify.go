package assistantjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// ReplyReadyKind is the notification's kind, in its data, and its event
// type. The client recognises a reply-ready notification by it.
const ReplyReadyKind = "assistant_reply_ready"

// replyReadySource names this activity as the notification's source.
const replyReadySource = "assistantjobs.NotifyUnseenTurn"

// threadRecordEntity is the record-link registry's name for a conversation.
const threadRecordEntity = "assistant_thread"

// deskPath is where a conversation is opened from when the record-link
// registry cannot name the conversation itself: the Desk lists it under
// "Where you left off".
const deskPath = "/desk"

// replyNotifications is what telling someone their reply is ready needs of
// notifications.
type replyNotifications interface {
	ExistsRecent(
		ctx context.Context,
		req repositories.ExistsRecentNotificationRequest,
	) (bool, error)
	Create(
		ctx context.Context,
		entity *notification.Notification,
	) (*notification.Notification, error)
}

// replyThreads is what telling someone their reply is ready needs of their
// conversations: reading the one the reply is in, and keeping it.
type replyThreads interface {
	GetThread(ctx context.Context, req repositories.GetThreadRequest) (*conversation.Thread, error)
	UpdateThread(
		ctx context.Context,
		req *serviceports.UpdateThreadRequest,
		actor *serviceports.RequestActor,
	) (*conversation.Thread, error)
}

// notifiesUnseen reports the endings worth telling someone about when they
// were not watching. A stopped reply is not one: they stopped it.
func notifiesUnseen(status conversation.AssistantTurnStatus) bool {
	switch status {
	case conversation.AssistantTurnStatusCompleted,
		conversation.AssistantTurnStatusRefused,
		conversation.AssistantTurnStatusFailed:
		return true
	case conversation.AssistantTurnStatusPending,
		conversation.AssistantTurnStatusRunning,
		conversation.AssistantTurnStatusStopped:
		return false
	default:
		return false
	}
}

// NotifyUnseenTurnActivity tells the person who asked that their reply ended
// while nobody was reading it.
//
// A retried attempt finds the notification an earlier one created and stops
// there, so nobody is told twice. A quick question's hidden conversation is
// kept first, so the notification leads somewhere the sweep of unkept quick
// questions will not delete.
func (a *Activities) NotifyUnseenTurnActivity(
	ctx context.Context,
	in *NotifyUnseenTurnInput,
) error {
	if a.notifications == nil || a.threads == nil {
		return nil
	}

	payload := in.Payload
	tenant := payload.tenantInfo()
	correlation := payload.TurnID.String()

	sent, err := a.notifications.ExistsRecent(ctx, repositories.ExistsRecentNotificationRequest{
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
		EventType:      ReplyReadyKind,
		CorrelationID:  correlation,
		Since:          payload.Timestamp,
	})
	if err != nil {
		return fmt.Errorf("check whether this reply was already announced: %w", err)
	}
	if sent {
		return nil
	}

	thread, err := a.threads.GetThread(ctx, repositories.GetThreadRequest{
		ID:         payload.ThreadID,
		UserID:     payload.Actor.UserID,
		TenantInfo: tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) || dberror.IsNotFoundError(err) {
			// The conversation was deleted while the reply was being written.
			// There is nothing left to lead anybody to.
			a.logger.Info("not announcing a reply whose conversation is gone",
				zap.String("turn", correlation),
			)

			return nil
		}

		return fmt.Errorf("read the conversation this reply is in: %w", err)
	}

	if !thread.Origin.Listed() {
		thread, err = a.threads.UpdateThread(ctx, &serviceports.UpdateThreadRequest{
			ThreadID:   thread.ID,
			TenantInfo: tenant,
			Keep:       true,
		}, &payload.Actor)
		if err != nil {
			return fmt.Errorf("keep the conversation this reply is in: %w", err)
		}
	}

	if _, err = a.notifications.Create(ctx, replyReadyNotification(in, thread)); err != nil {
		return fmt.Errorf("announce this reply: %w", err)
	}

	return nil
}

// replyReadyNotification is the notification for a reply nobody saw end.
func replyReadyNotification(
	in *NotifyUnseenTurnInput,
	thread *conversation.Thread,
) *notification.Notification {
	payload := in.Payload
	buID := payload.BusinessUnitID
	userID := payload.Actor.UserID
	correlation := payload.TurnID.String()
	title, message, priority := replyReadyWording(in.Status)

	return &notification.Notification{
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: &buID,
		TargetUserID:   &userID,
		Channel:        notification.ChannelUser,
		EventType:      ReplyReadyKind,
		Priority:       priority,
		Title:          title,
		Message:        message,
		Source:         replyReadySource,
		CorrelationID:  &correlation,
		Data: map[string]any{
			"kind":     ReplyReadyKind,
			"threadId": thread.ID.String(),
			"turnId":   correlation,
			"status":   string(in.Status),
			"link":     threadLink(thread.ID),
		},
		RelatedEntities: map[string]any{
			"assistantThreadId": thread.ID.String(),
			"assistantTurnId":   correlation,
		},
	}
}

// replyReadyWording says how the reply ended. It names neither the
// conversation nor what the reply said: a notification is announced on the
// organization's realtime channel, which every signed-in member's socket
// receives, so a title or a preview here would hand one person's conversation
// to everybody in the organization. The client names the conversation from
// the thread id, which it reads as its owner.
func replyReadyWording(
	status conversation.AssistantTurnStatus,
) (title, message string, priority notification.Priority) {
	if status == conversation.AssistantTurnStatusFailed {
		return "Your reply could not finish",
			"The assistant stopped before it finished. Open the conversation to see what was kept and ask again.",
			notification.PriorityHigh
	}

	return "Your reply is ready", "Open the conversation to read it.", notification.PriorityMedium
}

// threadLink is where the notification opens the conversation. It comes from
// the record-link registry, which every link the server hands out is built
// from.
func threadLink(threadID pulid.ID) string {
	if path, ok := productguide.RecordPath(threadRecordEntity, threadID.String()); ok {
		return path
	}

	return deskPath
}
