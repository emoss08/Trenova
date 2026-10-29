package agentmemorysubjects

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/pkg/pagination"
)

// SubjectLabeler names the record a memory is about, so the prompt can say
// "Acme Freight (customer)" rather than an id.
type SubjectLabeler interface {
	Label(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ref agentmemoryports.MemorySubjectRef,
	) (string, error)
}
