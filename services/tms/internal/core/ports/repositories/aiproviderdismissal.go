package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListFailureDismissalsRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

type DeleteFailureDismissalRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	ProviderID pulid.ID
}

type AIProviderFailureDismissalRepository interface {
	List(ctx context.Context, req *ListFailureDismissalsRequest) ([]*aiprovider.FailureDismissal, error)
	// Upsert keeps one dismissal per person and provider, the latest.
	Upsert(ctx context.Context, dismissal *aiprovider.FailureDismissal) error
	Delete(ctx context.Context, req *DeleteFailureDismissalRequest) error
}
