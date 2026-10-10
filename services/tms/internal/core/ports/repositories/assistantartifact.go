package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ListArtifactsRequest reads one page of a thread's artifacts by lineage,
// pinned lineages first, then the most recently added to.
type ListArtifactsRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	// Limit is how many lineages the page holds; zero means the repository's
	// default.
	Limit int
	// Cursor is the NextCursor of the page before; empty for the first.
	Cursor string
	// Query narrows to lineages whose title or tool contains it.
	Query string
	// Family narrows to one family of kinds; empty for all.
	Family assistantartifact.Family
	// PinnedOnly narrows to pinned lineages.
	PinnedOnly bool
	// Summary leaves each payload with only the keys that say what an
	// artifact is (assistantartifact.SummaryPayloadKeys); the full payload is
	// read with its lineage when it is shown.
	Summary bool
}

// ArtifactCounts is how many lineages match the query, in all, pinned and by
// family, for the browser's filter chips. They ignore the family and pinned
// filters, so every chip says what choosing it would show.
type ArtifactCounts struct {
	All      int                              `json:"all"`
	Pinned   int                              `json:"pinned"`
	Families map[assistantartifact.Family]int `json:"families"`
}

// ArtifactPage is one page of lineages with every version of each, the number
// of lineages that match in all, and where the next page starts.
type ArtifactPage struct {
	Artifacts  []*assistantartifact.Artifact
	Total      int
	NextCursor string
	Counts     ArtifactCounts
}

// LineageRequest names a lineage by any of its versions' ids.
type LineageRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type SlugRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Slug       string
}

type LatestInLineageRequest struct {
	ThreadID       pulid.ID
	TenantInfo     pagination.TenantInfo
	LineageKey     string
	ExceptToolCall string
}

type ListArtifactsByToolCallsRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	CallIDs    []string
}

type DeleteArtifactsRequest struct {
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
}

type GetArtifactRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type SetArtifactPinnedRequest struct {
	ID         pulid.ID
	ThreadID   pulid.ID
	TenantInfo pagination.TenantInfo
	Pinned     bool
}

// UpdateArtifactStatusRequest moves an artifact along, such as a draft that
// was sent or a run that finished, and may replace what it shows.
type UpdateArtifactStatusRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Status     assistantartifact.Status
	Payload    map[string]any
}

type AssistantArtifactRepository interface {
	// Upsert writes an artifact, replacing the one with the same thread,
	// source tool call and kind, or the same proposal or plan, when there is
	// one: a retried turn updates what it produced rather than adding a twin.
	Upsert(
		ctx context.Context,
		artifact *assistantartifact.Artifact,
	) (*assistantartifact.Artifact, error)
	ListPage(ctx context.Context, req ListArtifactsRequest) (*ArtifactPage, error)
	// ListLineage reads every version of the lineage an artifact belongs to,
	// oldest first.
	ListLineage(ctx context.Context, req LineageRequest) ([]*assistantartifact.Artifact, error)
	// FindBySlug reads the first version of the lineage a link names.
	FindBySlug(ctx context.Context, req SlugRequest) (*assistantartifact.Artifact, error)
	// TakenSlugs is the slugs a thread's lineages use that start with base.
	TakenSlugs(
		ctx context.Context,
		threadID pulid.ID,
		tenant pagination.TenantInfo,
		base string,
	) (map[string]bool, error)
	// InsertVersion adds a version a person made, which no tool call names.
	InsertVersion(
		ctx context.Context,
		artifact *assistantartifact.Artifact,
	) (*assistantartifact.Artifact, error)
	// ToolCallArguments is what a tool call in the thread was asked with, read
	// from the assistant message that made it; nil when no message holds it.
	ToolCallArguments(
		ctx context.Context,
		threadID pulid.ID,
		tenant pagination.TenantInfo,
		callID string,
	) (map[string]any, error)
	// TurnQuestions is, for each assistant message, the question the person
	// asked in the turn that wrote it.
	TurnQuestions(
		ctx context.Context,
		threadID pulid.ID,
		tenant pagination.TenantInfo,
		messageIDs []pulid.ID,
	) (map[pulid.ID]string, error)
	GetByID(ctx context.Context, req GetArtifactRequest) (*assistantartifact.Artifact, error)
	ListByToolCalls(
		ctx context.Context,
		req *ListArtifactsByToolCallsRequest,
	) ([]*assistantartifact.Artifact, error)
	Delete(ctx context.Context, req *DeleteArtifactsRequest) error
	// SetPinned pins or unpins the whole lineage the artifact belongs to and
	// returns its latest version.
	SetPinned(
		ctx context.Context,
		req SetArtifactPinnedRequest,
	) (*assistantartifact.Artifact, error)
	UpdateStatus(
		ctx context.Context,
		req UpdateArtifactStatusRequest,
	) (*assistantartifact.Artifact, error)
	LatestInLineage(
		ctx context.Context,
		req LatestInLineageRequest,
	) (*assistantartifact.Artifact, error)
	// FindByProposal reads the artifact that views a proposal, for the
	// status to follow the decision.
	FindByProposal(
		ctx context.Context,
		tenant pagination.TenantInfo,
		proposalID pulid.ID,
	) (*assistantartifact.Artifact, error)
}
