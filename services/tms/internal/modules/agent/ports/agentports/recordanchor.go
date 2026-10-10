package agentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type RecordAnchorRequest struct {
	TenantInfo pagination.TenantInfo
	Actor      *permissionports.RequestActor
	Records    []agent.EntityRef
	Timezone   string
}

type RecordAnchorReader interface {
	Anchor(ctx context.Context, req *RecordAnchorRequest) []agentdefinition.RuntimeAnchor
}
