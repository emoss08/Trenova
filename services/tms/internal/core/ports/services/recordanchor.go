package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/pagination"
)

type RecordAnchorRequest struct {
	TenantInfo pagination.TenantInfo
	Actor      *RequestActor
	Records    []agent.EntityRef
	Timezone   string
}

type RecordAnchorReader interface {
	Anchor(ctx context.Context, req *RecordAnchorRequest) []agentdefinition.RuntimeAnchor
}
