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

type NotificationRepository interface {
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
