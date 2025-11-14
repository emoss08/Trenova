package notification

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/sourcegraph/conc"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger                 *zap.Logger
	NotificationRepository repositories.NotificationRepository
	WebSocketService       services.WebSocketService
}

type Service struct {
	l                      *zap.Logger
	notificationRepository repositories.NotificationRepository
	webSocketService       services.WebSocketService
}

func NewService(p ServiceParams) services.NotificationService {
	return &Service{
		l:                      p.Logger.Named("service.notification"),
		notificationRepository: p.NotificationRepository,
		webSocketService:       p.WebSocketService,
	}
}

func (s *Service) SendNotification(
	ctx context.Context,
	req *services.SendNotificationRequest,
) error {
	s.l.Info("sending notification",
		zap.String("event_type", string(req.EventType)),
		zap.String("priority", string(req.Priority)),
		zap.String("channel", string(req.Targeting.Channel)),
	)

	notif := &notification.Notification{
		EventType:       req.EventType,
		Priority:        req.Priority,
		Channel:         req.Targeting.Channel,
		OrganizationID:  req.Targeting.OrganizationID,
		BusinessUnitID:  req.Targeting.BusinessUnitID,
		TargetUserID:    req.Targeting.TargetUserID,
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

	if err := s.notificationRepository.Create(ctx, notif); err != nil {
		s.l.Error("failed to create notification", zap.Error(err))
		return fmt.Errorf("failed to create notification: %w", err)
	}

	s.webSocketService.BroadcastToUser(ctx, notif.TargetUserID.String(), notif)

	now := utils.NowUnix()
	notif.DeliveryStatus = notification.DeliveryStatusDelivered
	notif.DeliveredAt = &now

	if err := s.notificationRepository.Update(ctx, notif); err != nil {
		s.l.Error("failed to update notification delivery status", zap.Error(err))
		// ! Don't return error here as the notification was already sent
	}

	s.l.Info("notification sent successfully",
		zap.String("notificationID", notif.ID.String()),
	)

	return nil
}

func (s *Service) SendJobCompletionNotification(
	ctx context.Context,
	req *services.JobCompletionNotificationRequest,
) error {
	s.l.Info("sending job completion notification",
		zap.String("job_id", req.JobID),
		zap.String("job_type", req.JobType),
		zap.Bool("success", req.Success),
	)

	eventType := GetEventType(req.JobType)
	priority := GetPriority(req.JobType, req.Success)
	title := GetTitle(req.JobType, req.Success)
	message := GetMessage(req.Success, req.JobType, req.JobID, req.Result)
	tags := GetTags(req.JobType)

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

func (s *Service) SendConfigurationCopiedNotification(
	ctx context.Context,
	req *services.ConfigurationCopiedNotificationRequest,
) error {
	s.l.Info("sending configuration copied notification",
		zap.Any("req", req),
	)

	notifReq := &services.SendNotificationRequest{
		EventType: notification.EventConfigurationCopied,
		Priority:  notification.PriorityLow,
		Targeting: notification.Targeting{
			Channel:        notification.ChannelUser,
			OrganizationID: req.OrganizationID,
			BusinessUnitID: &req.BusinessUnitID,
			TargetUserID:   &req.UserID,
		},
		Title: "Configuration Copied",
		Message: fmt.Sprintf(
			"Configuration '%s' has been copied by %s",
			req.ConfigName,
			req.ConfigCopiedBy,
		),
		Data: map[string]any{
			"configId":       req.ConfigID.String(),
			"configName":     req.ConfigName,
			"configCreator":  req.ConfigCreator,
			"configCopiedBy": req.ConfigCopiedBy,
		},
	}

	return s.SendNotification(ctx, notifReq)
}

func (s *Service) SendReportExportNotification(
	ctx context.Context,
	req *services.ReportExportNotificationRequest,
) error {
	s.l.Info("sending report export notification",
		zap.Any("req", req),
	)

	notifReq := &services.SendNotificationRequest{
		EventType: notification.EventJobReportExport,
		Priority:  notification.PriorityMedium,
		Targeting: notification.Targeting{
			Channel:        notification.ChannelUser,
			OrganizationID: req.OrganizationID,
			BusinessUnitID: &req.BusinessUnitID,
			TargetUserID:   &req.UserID,
		},
		Title: "Report Export Completed",
		Message: fmt.Sprintf(
			"Your %s export (%s) is ready for download with %d rows.",
			req.ReportType,
			req.ReportFormat,
			req.ReportRowCount,
		),
		Data: map[string]any{
			"reportId":       req.ReportID.String(),
			"reportType":     req.ReportType,
			"reportFormat":   req.ReportFormat,
			"reportRowCount": req.ReportRowCount,
			"reportSize":     req.ReportSize,
			"reportURL":      req.ReportURL,
		},
		RelatedEntities: []notification.RelatedEntity{
			{
				Type: "report",
				ID:   req.ReportID,
				Name: req.ReportName,
				URL:  req.ReportURL,
			},
		},
		Actions: []notification.Action{
			{
				ID:       "download_report",
				Label:    "Download Report",
				Type:     "link",
				Style:    "primary",
				Endpoint: req.ReportURL,
			},
		},
	}

	return s.SendNotification(ctx, notifReq)
}

func (s *Service) SendOwnershipTransferNotification(
	ctx context.Context,
	req *services.OwnershipTransferNotificationRequest,
) error {
	s.l.Debug("sending configuration copied notification",
		zap.Any("req", req),
	)

	notifReq := &services.SendNotificationRequest{
		EventType: notification.EventShipmentOwnershipTransferred,
		Priority:  notification.PriorityMedium,
		Targeting: notification.Targeting{
			Channel:        notification.ChannelUser,
			OrganizationID: req.OrgID,
			BusinessUnitID: &req.BuID,
			TargetUserID:   &req.TargetUserID,
		},
		Title: "Shipment Ownership Transferred",
		Message: fmt.Sprintf(
			"Shipment %s has been transferred from %s to you",
			req.ProNumber,
			req.OwnerName,
		),
		Data: map[string]any{
			"proNumber": req.ProNumber,
			"ownerName": req.OwnerName,
		},
	}

	return s.SendNotification(ctx, notifReq)
}

func (s *Service) SendShipmentHoldReleaseNotification(
	ctx context.Context,
	req *services.ShipmentHoldReleaseNotificationRequest,
) error {
	s.l.Debug("sending shipment hold release notification",
		zap.Any("req", req),
	)

	notifReq := &services.SendNotificationRequest{
		EventType: notification.EventShipmentHoldRelease,
		Priority:  notification.PriorityHigh,
		Targeting: notification.Targeting{
			Channel:        notification.ChannelUser,
			OrganizationID: req.OrgID,
			BusinessUnitID: &req.BuID,
			TargetUserID:   &req.TargetUserID,
		},
		Title: "Shipment Hold Released",
		Message: fmt.Sprintf(
			"Shipment %s has been released from hold by %s",
			req.ProNumber,
			req.ReleasedByName,
		),
		Data: map[string]any{
			"proNumber":      req.ProNumber,
			"releasedByName": req.ReleasedByName,
		},
	}

	return s.SendNotification(ctx, notifReq)
}

func (s *Service) SendCommentNotification(
	ctx context.Context,
	req *services.ShipmentCommentNotificationRequest,
) error {
	s.l.Info("sending comment notification",
		zap.Any("req", req),
	)

	title := s.buildNotificationTitle(req)

	notifReq := &services.SendNotificationRequest{
		EventType: notification.EventShipmentComment,
		Priority:  notification.PriorityLow,
		Targeting: notification.Targeting{
			Channel:        notification.ChannelUser,
			OrganizationID: req.OrganizationID,
			BusinessUnitID: &req.BusinessUnitID,
			TargetUserID:   &req.MentionedUserID,
		},
		Title: title,
		Message: fmt.Sprintf(
			"%s mentioned you in a comment",
			req.OwnerName,
		),
		Data: map[string]any{
			"commentId": req.CommentID.String(),
			"ownerName": req.OwnerName,
			"ownerId":   req.OwnerID.String(),
		},
	}

	return s.SendNotification(ctx, notifReq)
}

func (s *Service) buildNotificationTitle(
	req *services.ShipmentCommentNotificationRequest,
) (title string) {
	if req.OwnerID == req.MentionedUserID {
		title = "You mentioned yourself in a comment"
	} else {
		title = fmt.Sprintf("%s mentioned you in a comment", req.OwnerName)
	}

	return title
}

func (s *Service) SendBulkCommentNotifications(
	ctx context.Context,
	reqs []*services.ShipmentCommentNotificationRequest,
) error {
	if len(reqs) == 0 {
		return nil
	}

	s.l.Info("sending bulk comment notifications",
		zap.Int("count", len(reqs)),
	)

	var wg conc.WaitGroup
	var errors []error
	errorChan := make(chan error, len(reqs))

	for _, req := range reqs {
		wg.Go(func() {
			if err := s.SendCommentNotification(ctx, req); err != nil {
				errorChan <- err
			}
		})
	}

	wg.Wait()
	close(errorChan)

	for err := range errorChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		s.l.Error("failed to send some comment notifications",
			zap.Error(errors[0]),
			zap.Int("totalRequests", len(reqs)),
			zap.Int("failedRequests", len(errors)),
		)
		return fmt.Errorf("failed to send %d out of %d notifications", len(errors), len(reqs))
	}

	s.l.Info("all bulk comment notifications sent successfully",
		zap.Int("count", len(reqs)),
	)

	return nil
}

func (s *Service) MarkAsRead(ctx context.Context, req repositories.MarkAsReadRequest) error {
	s.l.Info("marking notification as read",
		zap.String("notification_id", req.NotificationID.String()),
		zap.String("user_id", req.UserID.String()),
	)

	if err := s.notificationRepository.MarkAsRead(ctx, req); err != nil {
		s.l.Error("failed to mark notification as read", zap.Error(err))
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return nil
}

func (s *Service) MarkAsDismissed(
	ctx context.Context,
	req repositories.MarkAsDismissedRequest,
) error {
	s.l.Info("marking notification as dismissed",
		zap.String("notification_id", req.NotificationID.String()),
		zap.String("user_id", req.UserID.String()),
	)

	if err := s.notificationRepository.MarkAsDismissed(ctx, req); err != nil {
		s.l.Error("failed to mark notification as dismissed", zap.Error(err))
		return fmt.Errorf("failed to mark notification as dismissed: %w", err)
	}

	return nil
}

func (s *Service) GetUserNotifications(
	ctx context.Context,
	req *repositories.GetUserNotificationsRequest,
) (*pagination.ListResult[*notification.Notification], error) {
	return s.notificationRepository.GetUserNotifications(ctx, req)
}

func (s *Service) GetUnreadCount(
	ctx context.Context,
	userID pulid.ID,
	organizationID pulid.ID,
) (int, error) {
	s.l.Info("getting unread notification count",
		zap.String("user_id", userID.String()),
		zap.String("organization_id", organizationID.String()),
	)

	count, err := s.notificationRepository.GetUnreadCount(ctx, userID, organizationID)
	if err != nil {
		s.l.Error("failed to get unread notification count", zap.Error(err))
		return 0, fmt.Errorf("failed to get unread notification count: %w", err)
	}

	return count, nil
}

func (s *Service) ReadAllNotifications(
	ctx context.Context,
	req repositories.ReadAllNotificationsRequest,
) error {
	s.l.Info("reading all notifications",
		zap.String("user_id", req.UserID.String()),
		zap.String("organization_id", req.OrgID.String()),
		zap.String("business_unit_id", req.BuID.String()),
	)

	if err := s.notificationRepository.ReadAllNotifications(ctx, req); err != nil {
		s.l.Error("failed to read all notifications", zap.Error(err))
		return fmt.Errorf("failed to read all notifications: %w", err)
	}

	return nil
}
