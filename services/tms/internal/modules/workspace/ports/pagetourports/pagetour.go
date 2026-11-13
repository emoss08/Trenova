package pagetourports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagetour"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ViewRequest struct {
	TenantInfo pagination.TenantInfo
	Tour       string
}

type Repository interface {
	List(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*pagetour.View, error)
	Exists(ctx context.Context, req *ViewRequest) (bool, error)
	CountForUser(ctx context.Context, tenantInfo pagination.TenantInfo) (int, error)
	Upsert(ctx context.Context, entity *pagetour.View) (*pagetour.View, error)
}
