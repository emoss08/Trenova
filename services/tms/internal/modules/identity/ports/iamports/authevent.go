package iamports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/pkg/pagination"
)

type AuthEventRepository interface {
	Create(ctx context.Context, event *iam.AuthEvent) error
	DeleteBefore(ctx context.Context, before int64) (int64, error)
}

type ListUserSignInsRequest struct {
	TenantInfo pagination.TenantInfo
	Limit      int
}

type AuthEventHistoryRepository interface {
	ListSignInsForUser(ctx context.Context, req ListUserSignInsRequest) ([]*iam.AuthEvent, error)
}
