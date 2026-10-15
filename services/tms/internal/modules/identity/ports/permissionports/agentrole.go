package permissionports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const PermissionCacheVariantAgentRole = "agent-role"

type AgentRoleSource interface {
	UnattendedRoleID(ctx context.Context, tenant pagination.TenantInfo, agentID pulid.ID) (pulid.ID, error)
}
