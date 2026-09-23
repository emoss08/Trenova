package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AgentSubjectExistsRequest struct {
	TenantInfo  pagination.TenantInfo
	SubjectType agent.SubjectType
	SubjectID   pulid.ID
}

// AgentSubjectRepository answers whether the record an agent names is one
// the tenant holds, before anything is filed against it.
type AgentSubjectRepository interface {
	Exists(ctx context.Context, req AgentSubjectExistsRequest) (bool, error)
}
