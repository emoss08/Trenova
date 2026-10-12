package notificationports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GetPreferenceRequest struct {
	TenantInfo pagination.TenantInfo
}

type MarkDigestSentRequest struct {
	TenantInfo pagination.TenantInfo
	SentAt     int64
}

type ListDigestNotificationsRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
	Until      int64
	Limit      int
}

type PreferenceRepository interface {
	Get(ctx context.Context, req GetPreferenceRequest) (*notification.Preference, bool, error)
	Upsert(ctx context.Context, entity *notification.Preference) (*notification.Preference, error)
	ListDigestSubscribers(ctx context.Context) ([]*notification.Preference, error)
	MarkDigestSent(ctx context.Context, req MarkDigestSentRequest) error
	ListDigestNotifications(
		ctx context.Context,
		req ListDigestNotificationsRequest,
	) ([]*notification.Notification, error)
}

type PreferenceService interface {
	Get(ctx context.Context, tenantInfo pagination.TenantInfo) (*notification.Preference, error)
	Update(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		entity *notification.Preference,
	) (*notification.Preference, error)
}
