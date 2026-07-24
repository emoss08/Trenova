package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
)

type AgentControlRepository interface {
	GetOrCreate(ctx context.Context, tenantInfo pagination.TenantInfo) (*tenant.AgentControl, error)
	Update(ctx context.Context, entity *tenant.AgentControl) (*tenant.AgentControl, error)
}
