package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetSubscriptionRequest struct {
	TenantInfo pagination.TenantInfo
}

type UpdateSubscriptionStatusRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Version    int64
	Status     subscription.Status
}

type ListDueSubscriptionsRequest struct {
	Now   int64
	Limit int
}

type CountSubscriptionsByStatusRequest struct {
	Statuses []subscription.Status
}

type SubscriptionRepository interface {
	GetByOrganization(
		ctx context.Context,
		req GetSubscriptionRequest,
	) (*subscription.Subscription, error)
	Create(
		ctx context.Context,
		entity *subscription.Subscription,
	) (*subscription.Subscription, error)
	UpdateStatus(
		ctx context.Context,
		req *UpdateSubscriptionStatusRequest,
	) (*subscription.Subscription, error)
	ListDue(
		ctx context.Context,
		req *ListDueSubscriptionsRequest,
	) ([]*subscription.Subscription, error)
	CountByStatus(ctx context.Context, req *CountSubscriptionsByStatusRequest) (int, error)
}
