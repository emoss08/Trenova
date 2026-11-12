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

// MemorySubjectCandidate is one record a name might mean.
type MemorySubjectCandidate struct {
	ID    pulid.ID
	Label string
	Code  string
}

// FindMemorySubjectsRequest asks which records of one kind a name could mean,
// so a memory can be kept about a customer the caller knows only by name.
type FindMemorySubjectsRequest struct {
	TenantInfo pagination.TenantInfo
	Type       agentmemory.MemorySubjectType
	Name       string
	Limit      int
}

type AgentMemorySubjectRepository interface {
	FindSubjectsByName(
		ctx context.Context,
		req *FindMemorySubjectsRequest,
	) ([]MemorySubjectCandidate, error)
	ListRecordLinks(
		ctx context.Context,
		req *ListMemoryRecordLinksRequest,
	) ([]MemoryRecordLink, error)
}
