package agentmemoryports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// MemoryRecordLink is one record another names: a shipment's customer, the
// location of one of its stops, a document's owner, the carrier an inbound
// message was matched to.
type MemoryRecordLink struct {
	From pulid.ID
	Kind agentmemory.MemoryRecordKind
	ID   pulid.ID
}

// ListMemoryRecordLinksRequest asks, for many records of one kind at once,
// which records each names. Links come back in the order a person reads the
// record: a shipment's customer, then its stops in order, then who is
// assigned, then its carriers.
type ListMemoryRecordLinksRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       agentmemory.MemoryRecordKind
	IDs        []pulid.ID
}

type AgentMemorySubjectRepository interface {
	ListRecordLinks(
		ctx context.Context,
		req *ListMemoryRecordLinksRequest,
	) ([]MemoryRecordLink, error)
}
