package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type NotificationService interface {
	// SendNotification sends a notification to the specified targets
	SendNotification(ctx context.Context, req *SendNotificationRequest) error

	// SendJobCompletionNotification sends a notification when a job completes
	SendJobCompletionNotification(ctx context.Context, req *JobCompletionNotificationRequest) error

	// MarkAsRead marks a notification as read
	MarkAsRead(ctx context.Context, req repositories.MarkAsReadRequest) error

	// MarkAsDismissed marks a notification as dismissed
	MarkAsDismissed(ctx context.Context, req repositories.MarkAsDismissedRequest) error

	// ReadAllNotifications reads all notifications for a user
	ReadAllNotifications(ctx context.Context, req repositories.ReadAllNotificationsRequest) error

	// GetUserNotifications retrieves notifications for a user
	GetUserNotifications(
		ctx context.Context,
		req *repositories.GetUserNotificationsRequest,
	) (*ports.ListResult[*notification.Notification], error)

	// GetUnreadCount gets the count of unread notifications for a user
	GetUnreadCount(ctx context.Context, userID pulid.ID, organizationID pulid.ID) (int, error)
}

type SendNotificationRequest struct {
	EventType       notification.EventType       `json:"eventType"`
	Priority        notification.Priority        `json:"priority"`
	Targeting       notification.Targeting       `json:"targeting"`
	Title           string                       `json:"title"`
	Message         string                       `json:"message"`
	Data            map[string]any               `json:"data,omitempty"`
	RelatedEntities []notification.RelatedEntity `json:"relatedEntities,omitempty"`
	Actions         []notification.Action        `json:"actions,omitempty"`
	ExpiresAt       *int64                       `json:"expiresAt,omitempty"`
	Source          string                       `json:"source"`
	JobID           *string                      `json:"jobId,omitempty"`
	CorrelationID   *string                      `json:"correlationId,omitempty"`
	Tags            []string                     `json:"tags,omitempty"`
}

type JobCompletionNotificationRequest struct {
	JobID           string                       `json:"jobId"`
	JobType         string                       `json:"jobType"`
	UserID          pulid.ID                     `json:"userId"`
	OrganizationID  pulid.ID                     `json:"organizationId"`
	BusinessUnitID  pulid.ID                     `json:"businessUnitId"`
	Success         bool                         `json:"success"`
	Result          string                       `json:"result"`
	Data            map[string]any               `json:"data,omitempty"`
	RelatedEntities []notification.RelatedEntity `json:"relatedEntities,omitempty"`
	Actions         []notification.Action        `json:"actions,omitempty"`
}
