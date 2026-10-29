package tenantports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/pkg/pagination"
)

type AgentControlRepository interface {
	GetOrCreate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*agentcontrol.AgentControl, error)
	Update(
		ctx context.Context,
		entity *agentcontrol.AgentControl,
	) (*agentcontrol.AgentControl, error)
}
