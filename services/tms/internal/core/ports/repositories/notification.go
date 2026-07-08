package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListNotificationsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type MarkNotificationsReadRequest struct {
	IDs        []pulid.ID
	TenantInfo pagination.TenantInfo
}

type ExistsRecentNotificationRequest struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	EventType      string   `json:"eventType"`
	CorrelationID  string   `json:"correlationId"`
	Since          int64    `json:"since"`
}

type NotificationRepository interface {
	ExistsRecent(ctx context.Context, req ExistsRecentNotificationRequest) (bool, error)
	Create(
		ctx context.Context,
		entity *notification.Notification,
	) (*notification.Notification, error)
	List(
		ctx context.Context,
		req *ListNotificationsRequest,
	) (*pagination.ListResult[*notification.Notification], error)
	MarkAsRead(ctx context.Context, req MarkNotificationsReadRequest) error
	MarkAllAsRead(ctx context.Context, userID pulid.ID, tenantInfo pagination.TenantInfo) error
	CountUnread(
		ctx context.Context,
		userID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (int64, error)
}
