package deskattentionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/desklens"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AttentionRequest struct {
	TenantInfo pagination.TenantInfo
	Lens       desklens.Lens
	Kinds      []watchtower.SourceKind
	Severities []watchtower.Severity
	First      int
	RecordIDs  []pulid.ID
	// Working keeps the rows with an item an agent is working on right now.
	Working bool
	// Mine narrows a reader who may see everyone's cases to their own and
	// the ones nobody owns.
	Mine bool
	// Search keeps the rows with an item whose title, summary or record
	// label holds these words.
	Search string
	// Unseen keeps the rows with an item the reader has not opened yet.
	Unseen bool
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

// KindCount is how many open items of one kind the reader may see, which a
// filter offers the kind by.
type KindCount struct {
	Kind  watchtower.SourceKind
	Label string
	Lens  desklens.Lens
	Count int
}

type Attention struct {
	Groups []*Group
	Counts LensCounts
	// Kinds counts the open items of each kind in the list's own scope,
	// most first, so a filter offers only what is there to find.
	Kinds   []KindCount
	HasMore bool
	SeenAt  int64
	// SeesEveryone says the reader may see everyone's cases, so the list
	// can offer only their own.
	SeesEveryone bool
}

type DeskAttentionService interface {
	Attention(
		ctx context.Context,
		req *AttentionRequest,
		actor *permissionports.RequestActor,
	) (*Attention, error)
}
