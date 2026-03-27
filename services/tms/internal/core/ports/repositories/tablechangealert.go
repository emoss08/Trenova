package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetTCASubscriptionByIDRequest struct {
	SubscriptionID pulid.ID
	TenantInfo     pagination.TenantInfo
}

type ListTCASubscriptionsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type FindMatchingTCASubscriptionsRequest struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	TableName      string
	Operation      string
	RecordID       string
}

type TCASubscriptionRepository interface {
	Create(ctx context.Context, entity *tablechangealert.TCASubscription) (*tablechangealert.TCASubscription, error)
	Update(ctx context.Context, entity *tablechangealert.TCASubscription) (*tablechangealert.TCASubscription, error)
	GetByID(ctx context.Context, req GetTCASubscriptionByIDRequest) (*tablechangealert.TCASubscription, error)
	List(ctx context.Context, req *ListTCASubscriptionsRequest) (*pagination.ListResult[*tablechangealert.TCASubscription], error)
	Delete(ctx context.Context, id pulid.ID, tenantInfo pagination.TenantInfo) error
	FindMatchingSubscriptions(ctx context.Context, req FindMatchingTCASubscriptionsRequest) ([]*tablechangealert.TCASubscription, error)
}

type TCAAllowlistRepository interface {
	List(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*tablechangealert.TCAAllowlistedTable, error)
	IsTableAllowed(ctx context.Context, tableName string, tenantInfo pagination.TenantInfo) (bool, error)
}
