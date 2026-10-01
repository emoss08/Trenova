package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/iam"
)

type AuthEventRepository interface {
	Create(ctx context.Context, event *iam.AuthEvent) error
	DeleteBefore(ctx context.Context, before int64) (int64, error)
}
