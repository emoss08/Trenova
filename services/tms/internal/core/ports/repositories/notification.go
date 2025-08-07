/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetUserNotificationsRequest struct {
	Filter     *ports.LimitOffsetQueryOptions
	UnreadOnly bool `query:"unreadOnly"`
}

type MarkAsReadRequest struct {
	NotificationID pulid.ID `json:"notificationId"`
	UserID         pulid.ID `json:"userId"`
	OrgID          pulid.ID `json:"orgId"`
	BuID           pulid.ID `json:"buId"`
}

type MarkAsDismissedRequest struct {
	NotificationID pulid.ID `json:"notificationId"`
	UserID         pulid.ID `json:"userId"`
	OrgID          pulid.ID `json:"orgId"`
	BuID           pulid.ID `json:"buId"`
}

type ReadAllNotificationsRequest struct {
	UserID pulid.ID `json:"userId"`
	OrgID  pulid.ID `json:"orgId"`
	BuID   pulid.ID `json:"buId"`
}

type NotificationRepository interface {
	// Create creates a new notification
	Create(ctx context.Context, notif *notification.Notification) error

	// Update updates an existing notification
	Update(ctx context.Context, notif *notification.Notification) error

	// GetByID retrieves a notification by ID
	GetByID(ctx context.Context, id pulid.ID) (*notification.Notification, error)

	// GetUserNotifications retrieves notifications for a user with pagination
	GetUserNotifications(
		ctx context.Context,
		req *GetUserNotificationsRequest,
	) (*ports.ListResult[*notification.Notification], error)

	// GetUnreadCount gets the count of unread notifications for a user
	GetUnreadCount(ctx context.Context, userID pulid.ID, organizationID pulid.ID) (int, error)

	// ReadAllNotifications reads all notifications for a user
	ReadAllNotifications(ctx context.Context, req ReadAllNotificationsRequest) error

	// MarkAsRead marks a notification as read
	MarkAsRead(ctx context.Context, req MarkAsReadRequest) error

	// MarkAsDismissed marks a notification as dismissed
	MarkAsDismissed(
		ctx context.Context,
		req MarkAsDismissedRequest,
	) error

	// MarkAsDelivered marks a notification as delivered
	MarkAsDelivered(ctx context.Context, notificationID pulid.ID, deliveredAt int64) error

	// GetPendingRetries gets notifications that failed delivery and can be retried
	GetPendingRetries(ctx context.Context, limit int) ([]*notification.Notification, error)

	// GetExpiredNotifications gets notifications that have expired
	GetExpiredNotifications(ctx context.Context, limit int) ([]*notification.Notification, error)

	// DeleteOldNotifications deletes notifications older than the specified timestamp
	DeleteOldNotifications(ctx context.Context, olderThan int64) error
}
