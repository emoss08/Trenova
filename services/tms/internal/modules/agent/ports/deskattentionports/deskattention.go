package deskattentionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AttentionRequest struct {
	TenantInfo pagination.TenantInfo
	Lens       watchtower.DeskLens
	Kinds      []watchtower.SourceKind
	Severities []watchtower.Severity
	First      int
	RecordIDs  []pulid.ID
	// Working keeps the rows with an item an agent is working on right now.
	Working bool
}

type Record struct {
	Kind     recordkind.Kind
	ID       pulid.ID
	Label    string
	Subtitle string
	Status   string
	Readable bool
}

type Group struct {
	Key      pulid.ID
	Record   *Record
	Severity watchtower.Severity
	DueAt    *int64
	DueLabel string
	Unseen   bool
	Items    []*watchtower.Item
	// Ownership is who the record's case belongs to and how the reader stands
	// to it; nil when the row is about no record, which has no case.
	Ownership *recordcase.Ownership
}

type LensCounts struct {
	Everything int
	Watchtower int
	Triage     int
	Decisions  int
}

type Attention struct {
	Groups  []*Group
	Counts  LensCounts
	HasMore bool
	SeenAt  int64
}

type DeskAttentionService interface {
	Attention(
		ctx context.Context,
		req *AttentionRequest,
		actor *permissionports.RequestActor,
	) (*Attention, error)
}
