package notificationports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListNotificationsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListNotificationConnectionRequest struct {
	Filter              *pagination.QueryOptions `json:"filter"`
	Cursor              pagination.CursorInfo    `json:"-"`
	NotificationColumns []string                 `json:"-"`
	Lens                notification.Lens        `json:"lens"`
	UnreadOnly          bool                     `json:"unreadOnly"`
	PersonalOnly        bool                     `json:"-"`
	Now                 int64                    `json:"-"`
}

type SnoozeNotificationsRequest struct {
	NotificationActionRequest

	Until int64
}

type DismissAllNotificationsRequest struct {
	TenantInfo   pagination.TenantInfo
	Lens         notification.Lens
	PersonalOnly bool
	Now          int64
}

type ResolveNotificationSubjectRequest struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	SubjectKey     string   `json:"subjectKey"`
	Resolution     string   `json:"resolution"`
	Now            int64    `json:"now"`
}

type NotificationCountsRequest struct {
	TenantInfo   pagination.TenantInfo
	PersonalOnly bool
	Now          int64
}

type NotificationCounts struct {
	ForYou       int64 `json:"forYou"`
	ForYouUnread int64 `json:"forYouUnread"`
	Team         int64 `json:"team"`
	TeamUnread   int64 `json:"teamUnread"`
	Later        int64 `json:"later"`
}

type NotificationActionRequest struct {
	IDs          []pulid.ID
	TenantInfo   pagination.TenantInfo
	PersonalOnly bool
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
	NotifiedUsers(ctx context.Context, req ExistsRecentNotificationRequest) ([]pulid.ID, error)
	Create(
		ctx context.Context,
		entity *notification.Notification,
	) (*notification.Notification, error)
	List(
		ctx context.Context,
		req *ListNotificationsRequest,
	) (*pagination.ListResult[*notification.Notification], error)
	ListConnection(
		ctx context.Context,
		req *ListNotificationConnectionRequest,
	) (*pagination.CursorListResult[*notification.Notification], error)
	MarkAsRead(ctx context.Context, req NotificationActionRequest) error
	MarkAsUnread(ctx context.Context, req NotificationActionRequest) error
	MarkAllAsRead(
		ctx context.Context,
		userID pulid.ID,
		tenantInfo pagination.TenantInfo,
		personalOnly bool,
	) error
	Dismiss(ctx context.Context, req NotificationActionRequest) error
	DismissAll(ctx context.Context, req DismissAllNotificationsRequest) error
	Restore(ctx context.Context, req NotificationActionRequest) error
	Snooze(ctx context.Context, req SnoozeNotificationsRequest) error
	Wake(ctx context.Context, req NotificationActionRequest) error
	ResolveSubject(ctx context.Context, req ResolveNotificationSubjectRequest) (int64, error)
	Counts(ctx context.Context, req NotificationCountsRequest) (*NotificationCounts, error)
	CountUnread(
		ctx context.Context,
		userID pulid.ID,
		tenantInfo pagination.TenantInfo,
		personalOnly bool,
	) (int64, error)
}
