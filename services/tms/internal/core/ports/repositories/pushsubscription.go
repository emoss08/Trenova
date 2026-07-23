package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type SavePushSubscriptionRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Endpoint   string                `json:"endpoint"`
	P256dh     string                `json:"p256dh"`
	Auth       string                `json:"auth"`
	UserAgent  string                `json:"userAgent"`
}

type PushSubscriptionRepository interface {
	Save(
		ctx context.Context,
		req *SavePushSubscriptionRequest,
	) (*notification.PushSubscription, error)
	ListByUser(
		ctx context.Context,
		userID pulid.ID,
	) ([]*notification.PushSubscription, error)
	DeleteByEndpoint(
		ctx context.Context,
		userID pulid.ID,
		endpoint string,
	) error
	DeleteByID(
		ctx context.Context,
		id pulid.ID,
	) error
}
