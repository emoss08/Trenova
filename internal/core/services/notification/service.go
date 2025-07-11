package notification

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger                 *logger.Logger
	NotificationRepository repositories.NotificationRepository
	WebSocketService       services.WebSocketService
}

type Service struct {
	l                      *zerolog.Logger
	notificationRepository repositories.NotificationRepository
	webSocketService       services.WebSocketService
}

func NewService(p ServiceParams) services.NotificationService {
	log := p.Logger.With().
		Str("service", "notification").
		Logger()

	return &Service{
		l:                      &log,
		notificationRepository: p.NotificationRepository,
		webSocketService:       p.WebSocketService,
	}
}

func (s *Service) SendNotification(
	ctx context.Context,
	req *services.SendNotificationRequest,
) error {
	s.l.Info().
		Str("event_type", string(req.EventType)).
		Str("priority", string(req.Priority)).
		Str("channel", string(req.Targeting.Channel)).
		Msg("sending notification")

	// Create notification entity
	notif := &notification.Notification{
		EventType:       req.EventType,
		Priority:        req.Priority,
		Channel:         req.Targeting.Channel,
		OrganizationID:  req.Targeting.OrganizationID,
		BusinessUnitID:  req.Targeting.BusinessUnitID,
		TargetUserID:    req.Targeting.TargetUserID,
		TargetRoleID:    req.Targeting.TargetRoleID,
		Title:           req.Title,
		Message:         req.Message,
		Data:            req.Data,
		RelatedEntities: req.RelatedEntities,
		Actions:         req.Actions,
		ExpiresAt:       req.ExpiresAt,
		Source:          req.Source,
		JobID:           req.JobID,
		CorrelationID:   req.CorrelationID,
		Tags:            req.Tags,
		DeliveryStatus:  notification.DeliveryStatusPending,
	}

	// Persist notification
	if err := s.notificationRepository.Create(ctx, notif); err != nil {
		s.l.Error().Err(err).Msg("failed to create notification")
		return fmt.Errorf("failed to create notification: %w", err)
	}

	// Mark as delivered before sending
	now := timeutils.NowUnix()
	notif.DeliveryStatus = notification.DeliveryStatusDelivered
	notif.DeliveredAt = &now

	// Update in database first
	if err := s.notificationRepository.Update(ctx, notif); err != nil {
		s.l.Error().Err(err).Msg("failed to update notification delivery status")
		return fmt.Errorf("failed to update notification delivery status: %w", err)
	}

	// Send via WebSocket with delivered status
	s.webSocketService.BroadcastToUser(notif.TargetUserID.String(), notif)

	s.l.Info().
		Str("notification_id", notif.ID.String()).
		Msg("notification sent successfully")

	return nil
}

func (s *Service) SendJobCompletionNotification(
	ctx context.Context,
	req *services.JobCompletionNotificationRequest,
) error {
	s.l.Info().
		Str("job_id", req.JobID).
		Str("job_type", req.JobType).
		Bool("success", req.Success).
		Msg("sending job completion notification")

	// Use registry to get notification details
	eventType := GetEventType(req.JobType)
	priority := GetPriority(req.JobType, req.Success)
	title := GetTitle(req.JobType, req.Success)
	message := GetMessage(req.Success, req.JobType, req.JobID, req.Result)
	tags := GetTags(req.JobType)

	// Create notification request
	notifReq := &services.SendNotificationRequest{
		EventType: eventType,
		Priority:  priority,
		Targeting: notification.Targeting{
			Channel:        notification.ChannelUser,
			OrganizationID: req.OrganizationID,
			BusinessUnitID: &req.BusinessUnitID,
			TargetUserID:   &req.UserID,
		},
		Title:           title,
		Message:         message,
		Data:            req.Data,
		RelatedEntities: req.RelatedEntities,
		Actions:         req.Actions,
		Source:          "job_service",
		JobID:           &req.JobID,
		Tags:            tags,
	}

	return s.SendNotification(ctx, notifReq)
}

func (s *Service) MarkAsRead(ctx context.Context, req repositories.MarkAsReadRequest) error {
	s.l.Info().
		Str("notification_id", req.NotificationID.String()).
		Str("user_id", req.UserID.String()).
		Msg("marking notification as read")

	if err := s.notificationRepository.MarkAsRead(ctx, req); err != nil {
		s.l.Error().Err(err).Msg("failed to mark notification as read")
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return nil
}

func (s *Service) MarkAsDismissed(
	ctx context.Context,
	req repositories.MarkAsDismissedRequest,
) error {
	s.l.Info().
		Str("notification_id", req.NotificationID.String()).
		Str("user_id", req.UserID.String()).
		Msg("marking notification as dismissed")

	if err := s.notificationRepository.MarkAsDismissed(ctx, req); err != nil {
		s.l.Error().Err(err).Msg("failed to mark notification as dismissed")
		return fmt.Errorf("failed to mark notification as dismissed: %w", err)
	}

	return nil
}

func (s *Service) GetUserNotifications(
	ctx context.Context,
	req *repositories.GetUserNotificationsRequest,
) (*ports.ListResult[*notification.Notification], error) {
	notifications, err := s.notificationRepository.GetUserNotifications(ctx, req)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to get user notifications")
		return nil, fmt.Errorf("failed to get user notifications: %w", err)
	}

	return notifications, nil
}

func (s *Service) GetUnreadCount(
	ctx context.Context,
	userID pulid.ID,
	organizationID pulid.ID,
) (int, error) {
	s.l.Info().
		Str("user_id", userID.String()).
		Str("organization_id", organizationID.String()).
		Msg("getting unread notification count")

	count, err := s.notificationRepository.GetUnreadCount(ctx, userID, organizationID)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to get unread notification count")
		return 0, fmt.Errorf("failed to get unread notification count: %w", err)
	}

	return count, nil
}

func (s *Service) ReadAllNotifications(
	ctx context.Context,
	req repositories.ReadAllNotificationsRequest,
) error {
	s.l.Info().
		Str("user_id", req.UserID.String()).
		Str("organization_id", req.OrgID.String()).
		Str("business_unit_id", req.BuID.String()).
		Msg("reading all notifications")

	if err := s.notificationRepository.ReadAllNotifications(ctx, req); err != nil {
		s.l.Error().Err(err).Msg("failed to read all notifications")
		return fmt.Errorf("failed to read all notifications: %w", err)
	}

	return nil
}
